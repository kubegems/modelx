package registry

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	"github.com/opencontainers/go-digest"
	"kubegems.io/modelx/pkg/errors"
	"kubegems.io/modelx/pkg/types"
)

var ErrRegistryStoreNotFound = stderrors.New("not found")

type BlobContent struct {
	ContentType     string
	ContentLength   int64
	ContentEncoding string
	Content         io.ReadCloser
}

type BlobResponse struct {
	RedirectLocation string
	Content          *BlobContent
}

type RegistryStore interface {
	GetGlobalIndex(ctx context.Context, search string) (types.Index, error)

	GetIndex(ctx context.Context, repository string, search string) (types.Index, error)
	RemoveIndex(ctx context.Context, repository string) error

	ExistsManifest(ctx context.Context, repository string, reference string) (bool, error)
	GetManifest(ctx context.Context, repository string, reference string) (*types.Manifest, error)
	PutManifest(ctx context.Context, repository string, reference string, contentType string, manifest types.Manifest) error
	DeleteManifest(ctx context.Context, repository string, reference string) error

	GetBlob(ctx context.Context, repository string, digest digest.Digest) (*BlobResponse, error)
	PutBlob(ctx context.Context, repository string, digest digest.Digest, content BlobContent) (*BlobResponse, error)
	ExistsBlob(ctx context.Context, repository string, digest digest.Digest) (bool, error)
}

func BlobDigestPath(repository string, d digest.Digest) string {
	return path.Join(repository, "blobs", d.Algorithm().String(), d.Hex())
}

func IndexPath(repository string) string {
	return path.Join(repository, RegistryIndexFileName)
}

func ManifestPath(repository string, reference string) string {
	return path.Join(repository, "manifests", reference)
}

func SplitManifestPath(in string) (string, string) {
	in = strings.TrimPrefix(in, "manifests")
	return path.Split(in)
}

func IsRegistryStoreNotNotFound(err error) bool {
	return stderrors.Is(err, ErrRegistryStoreNotFound)
}

type Registry struct {
	Store RegistryStore
}

func (s *Registry) HeadManifest(w http.ResponseWriter, r *http.Request) {
	name, reference := GetRepositoryReference(r)
	exist, err := s.Store.ExistsManifest(r.Context(), name, reference)
	if err != nil {
		if IsRegistryStoreNotNotFound(err) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			ResponseError(w, err)
		}
		return
	}
	if exist {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func (s *Registry) GetGlobalIndex(w http.ResponseWriter, r *http.Request) {
	index, err := s.Store.GetGlobalIndex(r.Context(), r.URL.Query().Get("search"))
	if err != nil {
		if IsRegistryStoreNotNotFound(err) {
			ResponseOK(w, types.Index{})
		} else {
			ResponseError(w, err)
		}
		return
	}
	ResponseOK(w, index)
}

func (s *Registry) GetIndex(w http.ResponseWriter, r *http.Request) {
	name, _ := GetRepositoryReference(r)
	index, err := s.Store.GetIndex(r.Context(), name, r.URL.Query().Get("search"))
	if err != nil {
		if IsRegistryStoreNotNotFound(err) {
			ResponseError(w, errors.NewIndexUnknownError(name))
		} else {
			ResponseError(w, err)
		}
		return
	}
	ResponseOK(w, index)
}

func (s *Registry) DeleteIndex(w http.ResponseWriter, r *http.Request) {
	name, _ := GetRepositoryReference(r)
	if err := s.Store.RemoveIndex(r.Context(), name); err != nil {
		if IsRegistryStoreNotNotFound(err) {
			ResponseError(w, errors.NewIndexUnknownError(name))
		} else {
			ResponseError(w, err)
		}
		return
	}
	ResponseOK(w, "ok")
}

func (s *Registry) GetManifest(w http.ResponseWriter, r *http.Request) {
	name, reference := GetRepositoryReference(r)
	manifest, err := s.Store.GetManifest(r.Context(), name, reference)
	if err != nil {
		if IsRegistryStoreNotNotFound(err) {
			ResponseError(w, errors.NewManifestUnknownError(reference))
		} else {
			ResponseError(w, err)
		}
		return
	}
	ResponseOK(w, manifest)
}

func (s *Registry) PutManifest(w http.ResponseWriter, r *http.Request) {
	name, reference := GetRepositoryReference(r)
	var manifest types.Manifest
	if err := json.NewDecoder(r.Body).Decode(&manifest); err != nil {
		ResponseError(w, errors.NewManifestInvalidError(err))
		return
	}
	contenttype := r.Header.Get("Content-Type")
	if err := s.Store.PutManifest(r.Context(), name, reference, contenttype, manifest); err != nil {
		ResponseError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *Registry) DeleteManifest(w http.ResponseWriter, r *http.Request) {
	name, reference := GetRepositoryReference(r)
	if err := s.Store.DeleteManifest(r.Context(), name, reference); err != nil {
		if IsRegistryStoreNotNotFound(err) {
			ResponseError(w, errors.NewManifestUnknownError(reference))
		} else {
			ResponseError(w, err)
		}
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func GetRepositoryReference(r *http.Request) (string, string) {
	vars := mux.Vars(r)
	return vars["name"], vars["reference"]
}

func (s *Registry) HeadBlob(w http.ResponseWriter, r *http.Request) {
	BlobDigestFun(w, r, func(ctx context.Context, repository string, digest digest.Digest) {
		ok, err := s.Store.ExistsBlob(r.Context(), repository, digest)
		if err != nil {
			ResponseError(w, err)
			return
		}
		if ok {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	})
}

// 如果客户端 包含 contentLength 则直接上传
// 如果客户端 不包含 contentLength 则返回一个 Location 后续上传至该地址
func (s *Registry) PutBlob(w http.ResponseWriter, r *http.Request) {
	BlobDigestFun(w, r, func(ctx context.Context, repository string, digest digest.Digest) {
		log := logr.FromContextOrDiscard(ctx).WithValues("action", "put-blob", "repository", repository, "digest", digest.String())
		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			ResponseError(w, errors.NewContentTypeInvalidError("empty"))
			return
		}
		content := BlobContent{
			ContentLength: r.ContentLength,
			ContentType:   contentType,
			Content:       r.Body,
		}
		result, err := s.Store.PutBlob(r.Context(), repository, digest, content)
		if err != nil {
			log.Error(err, "store put blob")
			ResponseError(w, err)
			return
		}
		if location := result.RedirectLocation; location != "" {
			w.Header().Set("Location", location)
			w.WriteHeader(http.StatusTemporaryRedirect)
		} else {
			w.WriteHeader(http.StatusCreated)
		}
	})
}

func (s *Registry) GetBlob(w http.ResponseWriter, r *http.Request) {
	BlobDigestFun(w, r, func(ctx context.Context, repository string, digest digest.Digest) {
		log := logr.FromContextOrDiscard(ctx).WithValues("action", "get-blob", "repository", repository, "digest", digest.String())
		result, err := s.Store.GetBlob(r.Context(), repository, digest)
		if err != nil {
			log.Error(err, "store get blob")
			if IsRegistryStoreNotNotFound(err) {
				ResponseError(w, errors.NewBlobUnknownError(digest))
			}
			ResponseError(w, err)
			return
		}
		if location := result.RedirectLocation; location != "" {
			w.Header().Add("Location", location)
			w.WriteHeader(http.StatusFound)
		} else {
			w.Header().Set("Content-Length", strconv.Itoa(int(result.Content.ContentLength)))
			w.Header().Set("Content-Type", result.Content.ContentType)
			w.Header().Set("Content-Encoding", result.Content.ContentEncoding)
			w.WriteHeader(http.StatusOK)

			io.Copy(w, result.Content.Content)
		}
		return
	})
}

func BlobDigestFun(w http.ResponseWriter, r *http.Request, fun func(ctx context.Context, repository string, digest digest.Digest)) {
	name, _ := GetRepositoryReference(r)
	digeststr := mux.Vars(r)["digest"]
	digest, err := digest.Parse(digeststr)
	if err != nil {
		ResponseError(w, errors.NewDigestInvalidError(digeststr))
		return
	}
	fun(r.Context(), name, digest)
}

func ParseDescriptor(r *http.Request) (types.Descriptor, error) {
	digeststr := mux.Vars(r)["digest"]
	digest, err := digest.Parse(digeststr)
	if err != nil {
		return types.Descriptor{}, errors.NewDigestInvalidError(digeststr)
	}
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		return types.Descriptor{}, errors.NewContentTypeInvalidError("empty")
	}
	descriptor := types.Descriptor{
		Digest:    digest,
		MediaType: contentType,
	}
	return descriptor, nil
}

func ParseAndCheckContentRange(header http.Header) (int64, int64, error) {
	contentRange, contentLength := header.Get("Content-Range"), header.Get("Content-Length")
	ranges := strings.Split(contentRange, "-")
	if len(ranges) != 2 {
		return -1, -1, errors.NewContentRangeInvalidError("invalid format")
	}
	start, err := strconv.ParseInt(ranges[0], 10, 64)
	if err != nil {
		return -1, -1, errors.NewContentRangeInvalidError("invalid start")
	}
	end, err := strconv.ParseInt(ranges[1], 10, 64)
	if err != nil {
		return -1, -1, errors.NewContentRangeInvalidError("invalid end")
	}
	if start > end {
		return -1, -1, errors.NewContentRangeInvalidError("start > end")
	}
	contentLen, err := strconv.ParseInt(contentLength, 10, 64)
	if err != nil {
		return -1, -1, errors.NewContentRangeInvalidError("invalid content length")
	}
	if contentLen != (end-start)+1 {
		return -1, -1, errors.NewContentRangeInvalidError("content length != (end-start)+1")
	}
	return start, end, nil
}
