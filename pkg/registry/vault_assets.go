package registry

import (
	"context"
	"math/big"
	"sync"
	"time"

	"github.com/go-logr/logr"
	lru "github.com/hashicorp/golang-lru"
	"k8s.io/apimachinery/pkg/util/wait"
	"src.cloudminds.com/blockchain/vault/vault-sdk-go/sdk/models"
)

type AssetCache struct {
	s     *VaultClient
	mu    sync.Mutex
	cache *lru.Cache
}

func NewAssetCache(cli *VaultClient) *AssetCache {
	cache, err := lru.New(100)
	if err != nil {
		panic(err)
	}
	return &AssetCache{s: cli, cache: cache}
}

func (c *AssetCache) InitSync(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	log := logr.FromContextOrDiscard(ctx)

	assets, err := c.s.vault.GetAssets(ctx, c.s.ServiceProjectAddress())
	if err != nil {
		return err
	}
	for _, asset := range assets.Items {
		if len(asset.Attribute) == 0 {
			continue
		}
		attrs, err := models.UnmarshalAssetAttribute(asset.Attribute)
		if err != nil {
			continue
		}
		defaultitem, ok := attrs.Items["default"]
		if !ok {
			continue
		}
		log.Info("add asset mapping", "name", defaultitem.Name, "asset id", asset.AssetId.String())
		c.cache.Add(defaultitem.Name, asset)
	}
	return nil
}

func (c *AssetCache) GetByName(ctx context.Context, name string, ownneraddress string) (*models.Asset, error) {
	log := logr.FromContextOrDiscard(ctx).WithName("asset cache").WithValues("name", name)
	if val, ok := c.cache.Get(name); ok {
		cacheasset := val.(*models.Asset)
		log.V(4).Info("cache hit", "asset id", cacheasset.AssetId.String())
		return cacheasset, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	// check twice
	if val, ok := c.cache.Get(name); ok {
		cacheasset := val.(*models.Asset)
		log.V(4).Info("cache hit", "asset id", cacheasset.AssetId.String())
		return cacheasset, nil
	}

	log.Info("cache miss, try found")
	// find
	assets, err := c.s.vault.GetAssets(ctx, c.s.ServiceProjectAddress())
	if err != nil {
		return nil, err
	}
	for _, existsasset := range assets.Items {
		if len(existsasset.Attribute) == 0 {
			continue
		}
		attrs, err := models.UnmarshalAssetAttribute(existsasset.Attribute)
		if err != nil {
			continue
		}
		defaultitem, ok := attrs.Items["default"]
		if !ok {
			continue
		}
		if defaultitem.Name != name {
			continue
		}
		// found
		log.Info("found", "asset id", existsasset.AssetId.String())
		c.cache.Add(defaultitem.Name, existsasset)
		return existsasset, nil
	}

	if ownneraddress == "" {
		return nil, ErrRegistryStoreNotFound
	}

	// create
	log.Info("not found,try create", "ownner", ownneraddress)
	resp, err := c.s.vault.MintAsset(ctx,
		c.s.registerTimeout,
		c.s.ServiceWallet(),
		c.s.ServiceProjectAddress(),
		ownneraddress,
	)
	if err != nil {
		return nil, err
	}
	var assetid *big.Int
	if resp.Status == models.TransactionExecPending {
		if err := wait.PollUntilWithContext(ctx, time.Second, func(ctx context.Context) (done bool, err error) {
			resultresp, err := c.s.vault.GetMintAssetResult(ctx, resp.TxHash)
			if err != nil {
				return false, err
			}
			if resultresp.Status == models.TransactionExecSuccess {
				assetid = big.NewInt(int64(*resultresp.AssetId))
				return true, nil
			}
			log.Info("pending", "tx", resp.TxHash)
			return false, nil
		}); err != nil {
			return nil, err
		}
	} else {
		assetid = big.NewInt(int64(*resp.AssetId))
		log.Info("created", "tx", resp.TxHash, "asset id", assetid.String())
	}
	if err := c.s.vault.SetAssetAttribute(ctx,
		c.s.ServiceWallet(),
		nil, // access grant
		c.s.ServiceProjectAddress(),
		assetid,
		&models.AssetAttributeItem{
			Name:        name,
			Description: name,
			Tags:        []string{"modelx"},
			ImageUri:    "-",
			Attributes:  map[string]interface{}{"key": name},
		},
	); err != nil {
		log.Error(err, "set asset attr")
		return nil, err
	}
	createdasset, err := c.s.vault.GetAssetById(ctx, c.s.ServiceProjectAddress(), assetid)
	if err != nil {
		log.Error(err, "get asset by id", "asset id", assetid.String())
		return nil, err
	}
	// cache it
	log.Info("add asset mapping", "name", name, "asset id", createdasset.AssetId.String())
	c.cache.Add(name, createdasset)
	return createdasset, nil
}
