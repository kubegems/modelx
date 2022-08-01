package client

import (
	"context"
	"net/http"

	"kubegems.io/modelx/pkg/types"
)

func GetManifest(ctx context.Context, reference Reference) (*types.Manifest, error) {
	remote := RegistryClient{
		Client: &http.Client{},
		Addr:   reference.Registry,
	}
	return remote.GetManifest(ctx, reference.Repository, reference.Version)
}

func GetIndex(ctx context.Context, reference Reference, search string) (*types.Index, error) {
	remote := RegistryClient{
		Client: &http.Client{},
		Addr:   reference.Registry,
	}

	if reference.Repository == "" {
		return remote.GetGlobalIndex(ctx, search)
	} else {
		return remote.GetIndex(ctx, reference.Repository, search)
	}
}
