package registry

import (
	"context"
	stderrors "errors"
	"io"
	"path"
	"strings"

	"github.com/opencontainers/go-digest"
	"kubegems.io/modelx/pkg/types"
)

var ErrRegistryStoreNotFound = stderrors.New("not found")

type BlobLocation types.BlobLocation

var (
	BlobLocationPurposeUpload   = types.BlobLocationPurposeUpload
	BlobLocationPurposeDownload = types.BlobLocationPurposeDownload
)

type BlobContent struct {
	ContentType     string
	ContentLength   int64
	ContentEncoding string
	Content         io.ReadCloser
}

type RegistryStore interface {
	GetGlobalIndex(ctx context.Context, search string) (types.Index, error)

	GetIndex(ctx context.Context, repository string, search string) (types.Index, error)
	RemoveIndex(ctx context.Context, repository string) error

	ExistsManifest(ctx context.Context, repository string, reference string) (bool, error)
	GetManifest(ctx context.Context, repository string, reference string) (*types.Manifest, error)
	PutManifest(ctx context.Context, repository string, reference string, contentType string, manifest types.Manifest) error
	DeleteManifest(ctx context.Context, repository string, reference string) error

	ListBlobs(ctx context.Context, repository string) ([]digest.Digest, error)
	GetBlob(ctx context.Context, repository string, digest digest.Digest) (*BlobContent, error)
	DeleteBlob(ctx context.Context, repository string, digest digest.Digest) error
	PutBlob(ctx context.Context, repository string, digest digest.Digest, content BlobContent) error
	ExistsBlob(ctx context.Context, repository string, digest digest.Digest) (bool, error)

	GetBlobLocation(ctx context.Context, repository string, digest digest.Digest,
		purpose string, properties map[string]string) (*BlobLocation, error)
}

func BlobDigestPath(repository string, d digest.Digest) string {
	if d == "" {
		d = ":"
	}
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
