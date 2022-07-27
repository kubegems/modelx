package registry

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/opencontainers/go-digest"
	"kubegems.io/modelx/pkg/errors"
	"kubegems.io/modelx/pkg/types"
)

type Registry struct {
	Manifest *RegistryStore
}

func (s *Registry) HeadManifest(w http.ResponseWriter, r *http.Request) {
	name, reference := GetRepositoryReference(r)
	exist, err := s.Manifest.Exists(r.Context(), name, reference)
	if err != nil {
		ResponseError(w, err)
		return
	}
	if exist {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func (s *Registry) GetGlobalIndex(w http.ResponseWriter, r *http.Request) {
	index, err := s.Manifest.GetGlobalIndex(r.Context(), r.URL.Query().Get("search"))
	if err != nil {
		ResponseError(w, err)
		return
	}
	ResponseOK(w, index)
}

func (s *Registry) GetIndex(w http.ResponseWriter, r *http.Request) {
	name, _ := GetRepositoryReference(r)
	index, err := s.Manifest.GetIndex(r.Context(), name, r.URL.Query().Get("search"))
	if err != nil {
		if IsStorageNotFound(err) {
			err = errors.NewIndexUnknownError(name)
		}
		ResponseError(w, err)
		return
	}
	ResponseOK(w, index)
}

func (s *Registry) GetManifest(w http.ResponseWriter, r *http.Request) {
	name, reference := GetRepositoryReference(r)
	manifest, err := s.Manifest.GetManifest(r.Context(), name, reference)
	if err != nil {
		ResponseError(w, err)
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
	if err := s.Manifest.PutManifest(r.Context(), name, reference, contenttype, manifest); err != nil {
		ResponseError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *Registry) DeleteManifest(w http.ResponseWriter, r *http.Request) {
	name, reference := GetRepositoryReference(r)
	if err := s.Manifest.DeleteManifest(r.Context(), name, reference); err != nil {
		ResponseError(w, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *Registry) PostUpload(w http.ResponseWriter, r *http.Request) {
}

func GetRepositoryReference(r *http.Request) (string, string) {
	vars := mux.Vars(r)
	return vars["name"], vars["reference"]
}

func (s *Registry) HeadBlob(w http.ResponseWriter, r *http.Request) {
	BlobDigestFun(w, r, func(ctx context.Context, repository string, digest digest.Digest) {
		ok, err := s.Manifest.ExistsBlob(r.Context(), repository, digest)
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
func (s *Registry) PostBlob(w http.ResponseWriter, r *http.Request) {
	s.PutBlob(w, r)
}

func (s *Registry) PutBlob(w http.ResponseWriter, r *http.Request) {
	repository, _ := GetRepositoryReference(r)
	desc, err := ParseDescriptor(r)
	if err != nil {
		ResponseError(w, err)
		return
	}
	if err := s.Manifest.PutBlob(r.Context(), repository, *desc, r.Body); err != nil {
		ResponseError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *Registry) GetBlob(w http.ResponseWriter, r *http.Request) {
	BlobDigestFun(w, r, func(ctx context.Context, repository string, digest digest.Digest) {
		location, err := s.Manifest.GetBlobURL(r.Context(), repository, digest)
		if err != nil {
			if !errors.IsErrCode(err, errors.ErrCodeUnsupported) {
				ResponseError(w, err)
				return
			}
			rc, err := s.Manifest.GetBlob(r.Context(), repository, digest)
			if err != nil {
				ResponseError(w, err)
				return
			}
			w.Header().Set("Content-Length", strconv.Itoa(int(rc.ContentLength)))
			w.Header().Set("Content-Type", rc.ContentType)
			w.WriteHeader(http.StatusOK)
			io.Copy(w, rc)
			return
		}
		w.Header().Add("Location", location)
		w.WriteHeader(http.StatusFound)
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

func ParseDescriptor(r *http.Request) (*types.Descriptor, error) {
	digeststr := mux.Vars(r)["digest"]
	digest, err := digest.Parse(digeststr)
	if err != nil {
		return nil, errors.NewDigestInvalidError(digeststr)
	}
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		return nil, errors.NewContentTypeInvalidError("empty")
	}
	descriptor := &types.Descriptor{
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
