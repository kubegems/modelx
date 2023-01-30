package registry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/opencontainers/go-digest"
	"golang.org/x/exp/slices"
	"golang.org/x/sync/errgroup"
	"kubegems.io/modelx/pkg/errors"
	"kubegems.io/modelx/pkg/types"
)

const RegistryIndexFileName = "index.json"

type S3Options struct {
	URL           string        `json:"url,omitempty"`
	Region        string        `json:"region,omitempty"`
	Buket         string        `json:"buket,omitempty"`
	AccessKey     string        `json:"accessKey,omitempty"`
	SecretKey     string        `json:"secretKey,omitempty"`
	PresignExpire time.Duration `json:"presignExpire,omitempty"`
}

func NewDefaultS3Options() *S3Options {
	return &S3Options{
		Buket:         "registry",
		URL:           "https://s3.amazonaws.com",
		AccessKey:     "",
		SecretKey:     "",
		PresignExpire: time.Hour,
		Region:        "",
	}
}

type FSRegistryStore struct {
	FS             FSProvider
	EnableRedirect bool
}

func NewFSRegistryStore(ctx context.Context, s3options *S3Options, enableredirect bool) (*FSRegistryStore, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(s3options.AccessKey, s3options.SecretKey, ""),
		),
		config.WithRegion(s3options.Region),
		config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(
				func(service, region string, options ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{URL: s3options.URL}, nil
				},
			),
		),
	)
	if err != nil {
		return nil, err
	}
	s3cli := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})
	storage := &S3StorageProvider{
		Bucket:  s3options.Buket,
		Client:  s3cli,
		Expire:  s3options.PresignExpire,
		Prefix:  "registry",
		PreSign: s3.NewPresignClient(s3cli),
	}
	store := &FSRegistryStore{
		FS:             storage,
		EnableRedirect: enableredirect,
	}
	if err := store.RefreshGlobalIndex(ctx); err != nil {
		return nil, err
	}
	return store, nil
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

func (m *FSRegistryStore) ExistsManifest(ctx context.Context, repository string, reference string) (bool, error) {
	if ok, err := m.FS.Exists(ctx, ManifestPath(repository, reference)); err != nil {
		return false, errors.NewInternalError(err)
	} else {
		return ok, nil
	}
}

func (m *FSRegistryStore) GetManifest(ctx context.Context, repository string, reference string) (*types.Manifest, error) {
	body, err := m.FS.Get(ctx, ManifestPath(repository, reference))
	if err != nil {
		if IsS3StorageNotFound(err) {
			return nil, errors.NewManifestUnknownError(reference)
		}
		return nil, errors.NewInternalError(err)
	}
	defer body.Close()

	manifest := &types.Manifest{}
	if err := json.NewDecoder(body).Decode(manifest); err != nil {
		return nil, errors.NewManifestInvalidError(err)
	}
	return manifest, nil
}

func (m *FSRegistryStore) PutManifest(ctx context.Context, repository string, reference string, contentType string, manifest types.Manifest) error {
	content, err := json.Marshal(manifest)
	if err != nil {
		return errors.NewManifestInvalidError(err)
	}
	storageContent := BlobContent{
		Content:       io.NopCloser(bytes.NewReader(content)),
		ContentLength: int64(len(content)),
		ContentType:   contentType,
	}
	if err := m.FS.Put(ctx, ManifestPath(repository, reference), storageContent); err != nil {
		return errors.NewInternalError(err)
	}
	if err := m.RefreshIndex(ctx, repository); err != nil {
		return errors.NewInternalError(err)
	}
	return nil
}

func (m *FSRegistryStore) DeleteManifest(ctx context.Context, repository string, reference string) error {
	if err := m.FS.Remove(ctx, ManifestPath(repository, reference), false); err != nil {
		return errors.NewInternalError(err)
	}
	return nil
}

// Gettypes.Index returns the types.Index for the given repository. if no manifests return an empty types.Index.
func (m *FSRegistryStore) GetIndex(ctx context.Context, repository string, search string) (types.Index, error) {
	body, err := m.FS.Get(ctx, IndexPath(repository))
	if err != nil {
		return types.Index{}, err
	}
	defer body.Close()

	var index types.Index
	if err := json.NewDecoder(body).Decode(&index); err != nil {
		return types.Index{}, err
	}
	if search != "" {
		searchregexp, err := regexp.Compile(search)
		if err != nil {
			return types.Index{}, errors.NewParameterInvalidError(fmt.Sprintf("search %s: %v", search, err))
		}
		indexies := []types.Descriptor{}
		for _, manifest := range index.Manifests {
			if searchregexp.MatchString(manifest.Name) {
				indexies = append(indexies, manifest)
			}
		}
		index.Manifests = indexies
	}

	return index, nil
}

func (m *FSRegistryStore) PutIndex(ctx context.Context, repository string, index types.Index) error {
	slices.SortFunc(index.Manifests, func(a, b types.Descriptor) bool {
		return strings.Compare(a.Name, b.Name) < 0
	})

	// use latest manifest annotations as index annotations
	for _, manifest := range index.Manifests {
		if manifest.Annotations == nil {
			continue
		}
		index.Annotations = manifest.Annotations
		break
	}

	content, err := json.Marshal(index)
	if err != nil {
		return errors.NewInternalError(err)
	}
	storageContent := BlobContent{
		Content:       io.NopCloser(bytes.NewReader(content)),
		ContentLength: int64(len(content)),
		ContentType:   MediaTypeModelIndexJson,
	}
	if err := m.FS.Put(ctx, IndexPath(repository), storageContent); err != nil {
		return errors.NewInternalError(err)
	}
	return nil
}

func (m *FSRegistryStore) RemoveIndex(ctx context.Context, repository string) error {
	// remove all manifests and blobs
	if err := m.FS.Remove(ctx, repository, true); err != nil {
		return errors.NewInternalError(err)
	}
	if err := m.RefreshIndex(ctx, repository); err != nil {
		return errors.NewInternalError(err)
	}
	return nil
}

func (m *FSRegistryStore) RefreshIndex(ctx context.Context, repository string) error {
	filemetas, err := m.FS.List(ctx, ManifestPath(repository, ""), false)
	if err != nil {
		return errors.NewInternalError(err)
	}

	eg := errgroup.Group{}
	manifests := sync.Map{}
	for _, meta := range filemetas {
		meta := meta
		eg.Go(func() error {
			manifest, err := m.GetManifest(ctx, repository, meta.Name)
			if err != nil {
				return err
			}
			desc := types.Descriptor{
				Name:        meta.Name,
				Modified:    meta.LastModified,
				Annotations: manifest.Annotations,
				Size: func() int64 {
					size := manifest.Config.Size
					for _, blob := range manifest.Blobs {
						size += blob.Size
					}
					return size
				}(),
			}
			manifests.Store(meta.Name, desc)
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return errors.NewInternalError(err)
	}

	index := types.Index{}

	manifests.Range(func(key, value any) bool {
		index.Manifests = append(index.Manifests, value.(types.Descriptor))
		return true
	})

	// save the index
	if len(index.Manifests) != 0 {
		if err := m.PutIndex(ctx, repository, index); err != nil {
			return errors.NewInternalError(err)
		}
	}
	// refresh global index
	if err := m.RefreshGlobalIndex(ctx); err != nil {
		return errors.NewInternalError(err)
	}
	return nil
}

func (m *FSRegistryStore) GetGlobalIndex(ctx context.Context, search string) (types.Index, error) {
	body, err := m.FS.Get(ctx, IndexPath(""))
	if err != nil {
		return types.Index{}, err
	}
	defer body.Close()

	var globalindex types.Index
	if err := json.NewDecoder(body).Decode(&globalindex); err != nil {
		return types.Index{}, err
	}
	if search != "" {
		searchregexp, err := regexp.Compile(search)
		if err != nil {
			return types.Index{}, errors.NewParameterInvalidError(fmt.Sprintf("search %s: %v", search, err))
		}
		indexies := []types.Descriptor{}
		for _, index := range globalindex.Manifests {
			if searchregexp.MatchString(index.Name) {
				indexies = append(indexies, index)
			}
		}
		globalindex.Manifests = indexies
	}
	return globalindex, nil
}

func (m *FSRegistryStore) PutGlobalIndex(ctx context.Context, index types.Index) error {
	slices.SortFunc(index.Manifests, types.SortDescriptorName)
	content, err := json.Marshal(index)
	if err != nil {
		return errors.NewInternalError(err)
	}
	storageContent := BlobContent{
		Content:       io.NopCloser(bytes.NewReader(content)),
		ContentLength: int64(len(content)),
		ContentType:   MediaTypeModelIndexJson,
	}
	if err := m.FS.Put(ctx, IndexPath(""), storageContent); err != nil {
		return errors.NewInternalError(err)
	}
	return nil
}

func (m *FSRegistryStore) RefreshGlobalIndex(ctx context.Context) error {
	filemetas, err := m.FS.List(ctx, "", true)
	if err != nil {
		return errors.NewInternalError(err)
	}

	eg := errgroup.Group{}

	// indexmap := map[string]types.Descriptor{}
	indexmap := sync.Map{}
	for _, meta := range filemetas {
		if meta.Name == RegistryIndexFileName || path.Base(meta.Name) != RegistryIndexFileName {
			continue
		}
		repository := path.Dir(meta.Name)
		eg.Go(func() error {
			index, err := m.GetIndex(ctx, repository, "")
			if err != nil {
				return err
			}

			desc := types.Descriptor{
				Name:        repository,
				MediaType:   MediaTypeModelIndexJson,
				Annotations: index.Annotations,
			}
			indexmap.Store(repository, desc)
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return errors.NewInternalError(err)
	}

	index := types.Index{}

	indexmap.Range(func(key, value any) bool {
		index.Manifests = append(index.Manifests, value.(types.Descriptor))
		return true
	})
	// save the index
	return m.PutGlobalIndex(ctx, index)
}

func (m *FSRegistryStore) ExistsBlob(ctx context.Context, repository string, digest digest.Digest) (bool, error) {
	if exists, err := m.FS.Exists(ctx, BlobDigestPath(repository, digest)); err != nil {
		return false, errors.NewInternalError(err)
	} else {
		return exists, nil
	}
}

func (m *FSRegistryStore) GetBlob(ctx context.Context, repository string, digest digest.Digest) (*BlobResponse, error) {
	path := BlobDigestPath(repository, digest)
	if m.EnableRedirect {
		location, err := m.FS.GetLocation(ctx, path)
		if err != nil {
			return nil, errors.NewInternalError(err)
		}
		return &BlobResponse{RedirectLocation: location}, nil
	} else {
		content, err := m.FS.Get(ctx, path)
		if err != nil {
			return nil, errors.NewInternalError(err)
		}
		return &BlobResponse{Content: &content}, nil
	}
}

func (m *FSRegistryStore) PutBlob(ctx context.Context, repository string, digest digest.Digest, content BlobContent) (*BlobResponse, error) {
	path := BlobDigestPath(repository, digest)
	if m.EnableRedirect {
		location, err := m.FS.PutLocation(ctx, path)
		if err != nil {
			return nil, errors.NewInternalError(err)
		}
		return &BlobResponse{RedirectLocation: location}, nil
	} else {
		if err := m.FS.Put(ctx, path, content); err != nil {
			return nil, errors.NewInternalError(err)
		} else {
			return &BlobResponse{}, nil
		}
	}
}
