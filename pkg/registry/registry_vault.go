package registry

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/opencontainers/go-digest"
	"kubegems.io/modelx/pkg/types"
	"src.cloudminds.com/blockchain/vault/vault-sdk-go/sdk/models"
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
		if len(asset.Attribute) == 0 {
			continue
		}
		attrs, err := models.UnmarshalAssetAttribute(asset.Attribute)
		if err != nil {
			continue
		}
		defaultattr, ok := attrs.Items["default"]
		if !ok {
			continue
		}
		if defaultattr.Name == "" {
			continue
		}
		globalindex.Manifests = append(globalindex.Manifests, types.Descriptor{
			Name: defaultattr.Name,
		})
	}
	return globalindex, nil
}

func (s *VaultRegistryStore) GetIndex(ctx context.Context, repository string, search string) (types.Index, error) {
	ret := types.Index{}
	assetid, err := s.getAssetID(ctx, repository)
	if err != nil {
		return ret, err
	}
	rawfiles, err := s.VaultClient.ListAssetFile(ctx, assetid, ManifestPath("", ""))
	if err != nil {
		return ret, err
	}
	for _, file := range rawfiles {
		_, versionname := SplitManifestPath(file.Name)
		ret.Manifests = append(ret.Manifests, types.Descriptor{
			Name:     versionname,
			Modified: time.UnixMicro(file.LastModified),
			Size:     0, // we can't calc blobs size
		})
	}
	return ret, nil
}

func (s *VaultRegistryStore) RemoveIndex(ctx context.Context, repository string) error {
	// TODO: nothing todo
	return nil
}

func (s *VaultRegistryStore) ExistsManifest(ctx context.Context, repository string, reference string) (bool, error) {
	assetid, err := s.getAssetID(ctx, repository)
	if err != nil {
		return false, nil
	}
	return s.VaultClient.HasAssetFile(ctx, assetid, ManifestPath("", reference))
}

func (s *VaultRegistryStore) GetManifest(ctx context.Context, repository string, reference string) (*types.Manifest, error) {
	assetid, err := s.getAssetID(ctx, repository)
	if err != nil {
		return nil, err
	}
	manifestdata, err := s.VaultClient.GetAssetFile(ctx, assetid, ManifestPath("", reference))
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
	assetid, err := s.getAssetID(ctx, repository)
	if err != nil {
		return err
	}
	return s.VaultClient.PutAssetFile(ctx, assetid, ManifestPath("", reference), manifestdata)
}

func (s *VaultRegistryStore) DeleteManifest(ctx context.Context, repository string, reference string) error {
	assetid, err := s.getAssetID(ctx, repository)
	if err != nil {
		return err
	}
	return s.VaultClient.RemoveAssetFile(ctx, assetid, ManifestPath("", reference))
}

func (s *VaultRegistryStore) GetBlob(ctx context.Context, repository string, digest digest.Digest) (*BlobResponse, error) {
	assetid, err := s.getAssetID(ctx, repository)
	if err != nil {
		return nil, err
	}
	redirecturl, err := s.VaultClient.GenerateBolbURL(ctx, assetid, BlobDigestPath("", digest), true, false)
	if err != nil {
		return nil, err
	}
	return &BlobResponse{RedirectLocation: redirecturl}, nil
}

func (s *VaultRegistryStore) PutBlob(ctx context.Context, repository string, digest digest.Digest, content BlobContent) (*BlobResponse, error) {
	assetid, err := s.getOrCreateAssetID(ctx, repository) // auto create
	if err != nil {
		return nil, err
	}
	redirecturl, err := s.VaultClient.GenerateBolbURL(ctx, assetid, BlobDigestPath("", digest), true, true)
	if err != nil {
		return nil, err
	}
	return &BlobResponse{RedirectLocation: redirecturl}, nil
}

func (s *VaultRegistryStore) ExistsBlob(ctx context.Context, repository string, digest digest.Digest) (bool, error) {
	assetid, err := s.getAssetID(ctx, repository)
	if err != nil {
		return false, err
	}
	return s.VaultClient.HasAssetFile(ctx, assetid, BlobDigestPath("", digest))
}

func (s *VaultRegistryStore) getOrCreateAssetID(ctx context.Context, repository string) (*big.Int, error) {
	ownaddress := s.VaultClient.serviceWallet.GetAddress()
	if !s.AssetsAlwaysOwnByServiceAddress {
		wallet, err := s.VaultClient.GetOrCreateWallet(ctx, UsernameFromContext(ctx))
		if err != nil {
			return nil, err
		}
		ownaddress = wallet.GetAddress()
	}
	asset, err := s.VaultClient.GetOrCreateAsset(ctx, repository, ownaddress)
	if err != nil {
		return nil, err
	}
	return asset.AssetId, nil
}

func (s *VaultRegistryStore) getAssetID(ctx context.Context, repository string) (*big.Int, error) {
	asset, err := s.VaultClient.GetAsset(ctx, repository)
	if err != nil {
		return nil, err
	}
	return asset.AssetId, nil
}
