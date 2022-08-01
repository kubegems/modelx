package registry

import (
	"context"
	"path"

	"github.com/opencontainers/go-digest"
	"kubegems.io/modelx/pkg/errors"
)

func (m *RegistryStore) ExistsBlob(ctx context.Context, repository string, digest digest.Digest) (bool, error) {
	if exists, err := m.Storage.Exists(ctx, BlobDigestPath(repository, digest)); err != nil {
		return false, errors.NewInternalError(err)
	} else {
		return exists, nil
	}
}

type BlobResponse struct {
	RedirectLocation string
	Content          StorageContent
}

func (m *RegistryStore) GetBlob(ctx context.Context, repository string, digest digest.Digest) (*BlobResponse, error) {
	path := BlobDigestPath(repository, digest)
	if m.EnableRedirect {
		location, err := m.Storage.GetLocation(ctx, path)
		if err != nil {
			return nil, errors.NewInternalError(err)
		}
		return &BlobResponse{RedirectLocation: location}, nil
	} else {
		content, err := m.Storage.Get(ctx, path)
		if err != nil {
			return nil, errors.NewInternalError(err)
		}
		return &BlobResponse{Content: content}, nil
	}
}

func (m *RegistryStore) PutBlob(ctx context.Context, repository string, digest digest.Digest, content StorageContent) (*BlobResponse, error) {
	path := BlobDigestPath(repository, digest)
	if m.EnableRedirect {
		location, err := m.Storage.PutLocation(ctx, path)
		if err != nil {
			return nil, errors.NewInternalError(err)
		}
		return &BlobResponse{RedirectLocation: location}, nil
	} else {
		if err := m.Storage.Put(ctx, path, content); err != nil {
			return nil, errors.NewInternalError(err)
		} else {
			return &BlobResponse{}, nil
		}
	}
}

func BlobDigestPath(repository string, d digest.Digest) string {
	return path.Join(repository, "blobs", d.Algorithm().String(), d.Hex())
}
