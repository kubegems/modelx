package registry

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"path"

	"kubegems.io/modelx/pkg/errors"
	"kubegems.io/modelx/pkg/types"
)

const (
	IndexFileName     = "index.json"
	AnnotationVersion = "modelx.model.version"
)

type RegistryStore struct {
	Storage *S3StorageProvider
}

func IsReference(reference string, descriptor types.Descriptor) bool {
	return descriptor.Annotations[AnnotationVersion] == reference || reference == descriptor.Digest.String()
}

func (m *RegistryStore) Exists(ctx context.Context, repository string, reference string) (bool, error) {
	if ok, err := m.Storage.Exists(ctx, ManifestPath(repository, reference)); err != nil {
		return false, errors.NewInternalError(err)
	} else {
		return ok, nil
	}
}

func (m *RegistryStore) GetManifest(ctx context.Context, repository string, reference string) (*types.Manifest, error) {
	body, err := m.Storage.Get(ctx, ManifestPath(repository, reference))
	if err != nil {
		if IsStorageNotFound(err) {
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

func (m *RegistryStore) PutManifest(ctx context.Context, repository string, reference string, contentType string, manifest types.Manifest) error {
	content, err := json.Marshal(manifest)
	if err != nil {
		return errors.NewManifestInvalidError(err)
	}
	storageContent := StorageContent{
		Content:       io.NopCloser(bytes.NewReader(content)),
		ContentLength: int64(len(content)),
		ContentType:   contentType,
	}
	if err := m.Storage.Put(ctx, ManifestPath(repository, reference), storageContent); err != nil {
		return errors.NewInternalError(err)
	}
	if err := m.RefreshIndex(ctx, repository); err != nil {
		return errors.NewInternalError(err)
	}
	return nil
}

func (m *RegistryStore) DeleteManifest(ctx context.Context, repository string, reference string) error {
	if err := m.Storage.Remove(ctx, ManifestPath(repository, reference)); err != nil {
		return errors.NewInternalError(err)
	}
	return nil
}

func ManifestPath(repository string, reference string) string {
	return path.Join(repository, "manifests", reference)
}
