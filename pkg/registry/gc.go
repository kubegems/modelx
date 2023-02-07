package registry

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/opencontainers/go-digest"
)

func GCBlobsAll(ctx context.Context, store RegistryStore) error {
	globalindex, err := store.GetGlobalIndex(ctx, "")
	if err != nil {
		return err
	}
	for _, repository := range globalindex.Manifests {
		if _, err := GCBlobs(ctx, store, repository.Name); err != nil {
			return err
		}
	}
	return nil
}

func GCBlobs(ctx context.Context, store RegistryStore, repository string) (map[digest.Digest]string, error) {
	log := logr.FromContextOrDiscard(ctx).WithValues("repository", repository)

	log.Info("star blobs garbage collect")
	defer log.Info("stop blobs garbage collect")

	manifests, err := store.GetIndex(ctx, repository, "")
	if err != nil {
		return nil, err
	}
	all, err := store.ListBlobs(ctx, repository)
	if err != nil {
		return nil, err
	}

	inuse := map[digest.Digest]struct{}{}
	for _, version := range manifests.Manifests {
		manifest, err := store.GetManifest(ctx, repository, version.Name)
		if err != nil {
			return nil, err
		}
		for _, blob := range append(manifest.Blobs, manifest.Config) {
			inuse[blob.Digest] = struct{}{}
		}
	}

	toremove := map[digest.Digest]string{}
	for _, blobdigest := range all {
		if _, ok := inuse[blobdigest]; !ok {
			log.WithValues("digest", blobdigest.String()).Info("mark blob unused")
			toremove[blobdigest] = ""
		}
	}

	for digest := range toremove {
		if err := store.DeleteBlob(ctx, repository, digest); err != nil {
			log.WithValues("digest", digest.String()).Error(err, "remove unused blob")
			toremove[digest] = err.Error()
			return nil, err
		} else {
			log.WithValues("digest", digest.String()).Info("removed unused blob")
			toremove[digest] = "removed"
		}
	}
	return toremove, nil
}
