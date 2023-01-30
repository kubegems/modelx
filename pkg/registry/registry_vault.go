package registry

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/opencontainers/go-digest"
	"kubegems.io/modelx/pkg/types"
)

type VaultRegistryStore struct {
	VaultClient                     *VaultClient
	AssetsAlwaysOwnByServiceAddress bool
}

func NewVaultRegistryStore(ctx context.Context, options *VaultOptions) (*VaultRegistryStore, error) {
	vaultcli, err := NewVaultClient(ctx, options)
	if err != nil {
		return nil, err
	}
	return &VaultRegistryStore{
		VaultClient:                     vaultcli,
		AssetsAlwaysOwnByServiceAddress: true, // modelx service account always owns all assets
	}, nil
}

func (s *VaultRegistryStore) GetGlobalIndex(ctx context.Context, search string) (types.Index, error) {
	assets, err := s.VaultClient.ListAssets(ctx)
	if err != nil {
		return types.Index{}, err
	}
	globalindex := types.Index{}
	for _, asset := range assets {
		globalindex.Manifests = append(globalindex.Manifests, types.Descriptor{
			Name: string(asset.Attribute),
		})
	}
	return globalindex, nil
}

func (s *VaultRegistryStore) GetIndex(ctx context.Context, repository string, search string) (types.Index, error) {
	ret := types.Index{}
	assetid, err := s.getRepositoryAssetID(ctx, repository, true)
	if err != nil {
		return ret, err
	}
	rawfiles, err := s.VaultClient.ListAssetFile(ctx, assetid, ManifestPath(repository, ""))
	if err != nil {
		return ret, err
	}
	for _, file := range rawfiles {
		ret.Manifests = append(ret.Manifests, types.Descriptor{
			Name:     file.Name,
			Modified: time.UnixMicro(file.LastModified),
			Size:     file.Size,
		})
	}
	return ret, nil
}

func (s *VaultRegistryStore) RemoveIndex(ctx context.Context, repository string) error {
	// TODO: nothing todo
	return nil
}

func (s *VaultRegistryStore) ExistsManifest(ctx context.Context, repository string, reference string) (bool, error) {
	assetid, err := s.getRepositoryAssetID(ctx, repository, false)
	if err != nil {
		return false, nil
	}
	return s.VaultClient.HasAssetFile(ctx, assetid, ManifestPath(repository, reference))
}

func (s *VaultRegistryStore) GetManifest(ctx context.Context, repository string, reference string) (*types.Manifest, error) {
	assetid, err := s.getRepositoryAssetID(ctx, repository, true)
	if err != nil {
		return nil, err
	}
	manifestdata, err := s.VaultClient.GetAssetFile(ctx, assetid, ManifestPath(repository, reference))
	if err != nil {
		return nil, err
	}
	manifest := &types.Manifest{}
	if err := json.Unmarshal(manifestdata, manifest); err != nil {
		return nil, err
	}
	return manifest, nil
}

func (s *VaultRegistryStore) PutManifest(ctx context.Context, repository string, reference string, contentType string, manifest types.Manifest) error {
	manifestdata, err := json.Marshal(manifest)
	if err != nil {
		return err
	}

	assetid, err := s.getRepositoryAssetID(ctx, repository, true)
	if err != nil {
		return err
	}
	return s.VaultClient.PutAssetFile(ctx, assetid, ManifestPath(repository, reference), manifestdata)
}

func (s *VaultRegistryStore) DeleteManifest(ctx context.Context, repository string, reference string) error {
	assetid, err := s.getRepositoryAssetID(ctx, repository, true)
	if err != nil {
		return err
	}
	return s.VaultClient.RemoveAssetFile(ctx, assetid, ManifestPath(repository, reference))
}

func (s *VaultRegistryStore) GetBlob(ctx context.Context, repository string, digest digest.Digest) (*BlobResponse, error) {
	assetid, err := s.getRepositoryAssetID(ctx, repository, true)
	if err != nil {
		return nil, err
	}
	redirecturl, err := s.VaultClient.GenerateBolbURL(ctx, assetid, true, false)
	if err != nil {
		return nil, err
	}
	return &BlobResponse{RedirectLocation: redirecturl}, nil
}

func (s *VaultRegistryStore) PutBlob(ctx context.Context, repository string, digest digest.Digest, content BlobContent) (*BlobResponse, error) {
	assetid, err := s.getRepositoryAssetID(ctx, repository, true)
	if err != nil {
		return nil, err
	}
	redirecturl, err := s.VaultClient.GenerateBolbURL(ctx, assetid, true, true)
	if err != nil {
		return nil, err
	}
	return &BlobResponse{RedirectLocation: redirecturl}, nil
}

func (s *VaultRegistryStore) ExistsBlob(ctx context.Context, repository string, digest digest.Digest) (bool, error) {
	assetid, err := s.getRepositoryAssetID(ctx, repository, true)
	if err != nil {
		return false, err
	}
	return s.VaultClient.HasAssetFile(ctx, assetid, BlobDigestPath(repository, digest))
}

func (s *VaultRegistryStore) getRepositoryAssetID(ctx context.Context, repository string, createOnNotFound bool) (*big.Int, error) {
	ownaddress := s.VaultClient.serviceWallet.GetAddress()
	if !s.AssetsAlwaysOwnByServiceAddress {
		wallet, err := s.VaultClient.GetOrCreateWallet(ctx, UsernameFromContext(ctx))
		if err != nil {
			return nil, err
		}
		ownaddress = wallet.GetAddress()
	}
	asset, _, err := s.VaultClient.GetOrCreateAsset(ctx, repository, ownaddress)
	if err != nil {
		return nil, err
	}
	return asset.AssetId, nil
}
