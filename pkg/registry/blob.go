package registry

import (
	"context"
	"io"
	"path"

	"github.com/opencontainers/go-digest"
	"kubegems.io/modelx/pkg/errors"
	"kubegems.io/modelx/pkg/types"
)

func (m *RegistryStore) ExistsBlob(ctx context.Context, repository string, digest digest.Digest) (bool, error) {
	if exists, err := m.Storage.Exists(ctx, BlobDigestPath(repository, digest)); err != nil {
		return false, errors.NewInternalError(err)
	} else {
		return exists, nil
	}
}

func (m *RegistryStore) GetBlob(ctx context.Context, repository string, digest digest.Digest) (StorageContent, error) {
	if reader, err := m.Storage.Get(ctx, BlobDigestPath(repository, digest)); err != nil {
		return StorageContent{}, errors.NewInternalError(err)
	} else {
		return reader, nil
	}
}

func (m *RegistryStore) PutBlob(ctx context.Context, repository string, desc types.Descriptor, content io.ReadCloser) error {
	if err := m.Storage.Put(ctx, BlobDigestPath(repository, desc.Digest), StorageContent{
		Content:     content,
		ContentType: desc.MediaType,
	}); err != nil {
		return errors.NewInternalError(err)
	}
	return nil
}

func (m *RegistryStore) GetBlobURL(ctx context.Context, repository string, digest digest.Digest) (string, error) {
	location, err := m.Storage.GetURL(ctx, BlobDigestPath(repository, digest))
	if err != nil {
		return "", errors.NewInternalError(err)
	}
	return location, nil
}

func (m *RegistryStore) UploadBlobURL(ctx context.Context, repository string, digest digest.Digest) (string, error) {
	location, err := m.Storage.UploadURL(ctx, BlobDigestPath(repository, digest))
	if err != nil {
		return "", errors.NewInternalError(err)
	}
	return location, nil
}

func BlobDigestPath(repository string, d digest.Digest) string {
	return path.Join(repository, "blobs", d.Algorithm().String(), d.Hex())
}
