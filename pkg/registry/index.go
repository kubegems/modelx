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

	"golang.org/x/exp/slices"
	"golang.org/x/sync/errgroup"
	"kubegems.io/modelx/pkg/errors"
	"kubegems.io/modelx/pkg/types"
)

// Gettypes.Index returns the types.Index for the given repository. if no manifests return an empty types.Index.
func (m *RegistryStore) GetIndex(ctx context.Context, repository string, search string) (types.Index, error) {
	body, err := m.Storage.Get(ctx, IndexPath(repository))
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

func (m *RegistryStore) PutIndex(ctx context.Context, repository string, index types.Index) error {
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
	storageContent := StorageContent{
		Content:       io.NopCloser(bytes.NewReader(content)),
		ContentLength: int64(len(content)),
		ContentType:   types.MediaTypeModelIndexJson,
	}
	if err := m.Storage.Put(ctx, IndexPath(repository), storageContent); err != nil {
		return errors.NewInternalError(err)
	}
	return nil
}

func (m *RegistryStore) RefreshIndex(ctx context.Context, repository string) error {
	filemetas, err := m.Storage.List(ctx, ManifestPath(repository, ""), false)
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
	if err := m.PutIndex(ctx, repository, index); err != nil {
		return errors.NewInternalError(err)
	}
	// refresh global index
	if err := m.RefreshGlobalIndex(ctx); err != nil {
		return errors.NewInternalError(err)
	}
	return nil
}

func (m *RegistryStore) GetGlobalIndex(ctx context.Context, search string) (types.Index, error) {
	body, err := m.Storage.Get(ctx, IndexPath(""))
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

func (m *RegistryStore) PutGlobalIndex(ctx context.Context, index types.Index) error {
	slices.SortFunc(index.Manifests, types.SortDescriptorName)
	content, err := json.Marshal(index)
	if err != nil {
		return errors.NewInternalError(err)
	}
	storageContent := StorageContent{
		Content:       io.NopCloser(bytes.NewReader(content)),
		ContentLength: int64(len(content)),
		ContentType:   types.MediaTypeModelIndexJson,
	}
	if err := m.Storage.Put(ctx, IndexPath(""), storageContent); err != nil {
		return errors.NewInternalError(err)
	}
	return nil
}

func (m *RegistryStore) RefreshGlobalIndex(ctx context.Context) error {
	filemetas, err := m.Storage.List(ctx, "", true)
	if err != nil {
		return errors.NewInternalError(err)
	}

	eg := errgroup.Group{}

	// indexmap := map[string]types.Descriptor{}
	indexmap := sync.Map{}
	for _, meta := range filemetas {
		if meta.Name == types.RegistryIndexFileName || path.Base(meta.Name) != types.RegistryIndexFileName {
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
				MediaType:   types.MediaTypeModelIndexJson,
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

func IndexPath(repository string) string {
	return path.Join(repository, types.RegistryIndexFileName)
}
