package client

import (
	"context"

	"kubegems.io/modelx/pkg/types"
)

type Client struct {
	remote RegistryClient
}

func NewClient(registry string, auth string) *Client {
	return &Client{
		remote: RegistryClient{Registry: registry, Authorization: auth},
	}
}

func (c Client) Ping(ctx context.Context) error {
	if _, err := c.remote.GetGlobalIndex(ctx, ""); err != nil {
		return err
	}
	return nil
}

func (c Client) GetManifest(ctx context.Context, repo, version string) (*types.Manifest, error) {
	return c.remote.GetManifest(ctx, repo, version)
}

func (c Client) PutManifest(ctx context.Context, repo, version string, manifest types.Manifest) error {
	return c.remote.PutManifest(ctx, repo, version, manifest)
}

func (c Client) GetIndex(ctx context.Context, repo string, search string) (*types.Index, error) {
	return c.remote.GetIndex(ctx, repo, search)
}

func (c Client) GetGlobalIndex(ctx context.Context, search string) (*types.Index, error) {
	return c.remote.GetGlobalIndex(ctx, search)
}
