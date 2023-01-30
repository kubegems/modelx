package registry

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/syndtr/goleveldb/leveldb"
	"k8s.io/apimachinery/pkg/util/wait"
	"kubegems.io/modelx/pkg/types"
	sdkerrs "src.cloudminds.com/blockchain/vault/vault-sdk-go/sdk/errs"
	"src.cloudminds.com/blockchain/vault/vault-sdk-go/sdk/models"
	sdkvault "src.cloudminds.com/blockchain/vault/vault-sdk-go/sdk/vault"
	sdkwallet "src.cloudminds.com/blockchain/vault/vault-sdk-go/sdk/wallet"
)

const DefaultRegisterTimeout = 30 * time.Second

type VaultOptions struct {
	Address  string `json:"address,omitempty"`
	Username string `json:"username,omitempty"`
	Mnemonic string `json:"mnemonic,omitempty"`
	Project  string `json:"project,omitempty"`
	Database string `json:"database,omitempty"`
}

func NewDefaultVaultOptions() *VaultOptions {
	return &VaultOptions{
		Address:  "",
		Username: "modelx",
		Project:  "modelx",
		Mnemonic: "",
		Database: "data/vault-db",
	}
}

type VaultClient struct {
	kv              KVDB
	vault           *sdkvault.Vault
	serviceWallet   *sdkwallet.Wallet
	serviceProject  *models.Project
	registerTimeout time.Duration
	vaultServiceUrl string
}

func NewVaultClient(ctx context.Context, options *VaultOptions) (*VaultClient, error) {
	vault, err := sdkvault.NewVault(ctx, options.Address)
	if err != nil {
		return nil, err
	}
	kv, err := NewLocalKVStore(options.Database)
	if err != nil {
		return nil, err
	}
	serviceWallet, err := sdkwallet.NewWallet(options.Username, options.Mnemonic, "")
	if err != nil {
		return nil, err
	}
	if _, err := vault.GetAccountByUsername(ctx, serviceWallet.GetUserName()); err != nil {
		return nil, fmt.Errorf("query service account: %w", err)
	}
	serviceProject, err := initProject(ctx, vault, serviceWallet, options)
	if err != nil {
		return nil, err
	}
	return &VaultClient{
		kv:              kv,
		vault:           vault,
		registerTimeout: DefaultRegisterTimeout,
		serviceWallet:   serviceWallet,
		serviceProject:  serviceProject,
		vaultServiceUrl: options.Address,
	}, nil
}

func initProject(ctx context.Context, vault *sdkvault.Vault, wallet *sdkwallet.Wallet, options *VaultOptions) (*models.Project, error) {
	log := logr.FromContextOrDiscard(ctx).WithValues("project", options.Project)

	projects, err := vault.GetProjects(ctx,
		sdkvault.ProjectsOwnerAddressOption(wallet.GetAddress()),
		sdkvault.ProjectsNameOption(options.Project),
	)
	if err != nil {
		return nil, err
	}
	projectaddress := ""
	if len(projects.Items) != 0 {
		for _, existsproject := range projects.Items {
			if existsproject.Name == options.Project {
				log.Info("exists project")
				return &existsproject, nil
			}
		}
	}

	log.Info("create project")
	// create project
	resp, err := vault.CreateGeneralProject(ctx, DefaultRegisterTimeout,
		wallet,                            // wallet
		options.Project,                   // name
		options.Project,                   // symbol
		big.NewInt(0),                     // supply
		models.ProjectEncMethodServerSide, // enc method
		models.SchemaString{Name: "key", MinLength: 1, MaxLength: 30}, // schema
	)
	if err != nil {
		return nil, err
	}
	// check transcation until success
	if resp.Status == models.TransactionExecPending {
		if err := wait.PollUntilWithContext(ctx, time.Second, func(ctx context.Context) (done bool, err error) {
			resp, err := vault.GetCreateGeneralProjectResult(ctx, resp.TxHash)
			if err != nil {
				return false, err
			}
			if resp.Status == models.TransactionExecSuccess {
				projectaddress = *resp.ProjectAddress
				return done, nil
			}
			log.Info("waiting project transcation")
			return false, nil
		}); err != nil {
			return nil, err
		}
	} else {
		projectaddress = *resp.ProjectAddress
	}
	log.Info("created project", "address", projectaddress)
	return vault.GetProjectByAddress(ctx, projectaddress)
}

func (s *VaultClient) ServiceProjectAddress() string {
	return s.serviceProject.Address.String()
}

func (s *VaultClient) GenerateBolbURL(ctx context.Context, assetid *big.Int, allowread, allowwrite bool) (string, error) {
	accessgrant, err := s.vault.ShareAssetRaw(ctx,
		s.ServiceWallet(),
		nil, // access grant
		&models.AssetShareList{Assets: []models.AssetList{{
			AssetIds:       []*big.Int{assetid},
			ProjectAddress: s.ServiceProjectAddress(),
		}}},
		sdkvault.AssetShareDisallowReadsOption(!allowread),
		sdkvault.AssetShareDisallowWritesOption(!allowwrite),
	)
	if err != nil {
		return "", err
	}
	vaultassetmeta := types.VaultBlobMeta{
		ServiceUrl:     s.vaultServiceUrl,
		ProjectAddress: s.ServiceProjectAddress(),
		AssetID:        assetid,
		AccessGrant:    accessgrant,
		Username:       UsernameFromContext(ctx),
	}
	return vaultassetmeta.ToURL()
}

func (s *VaultClient) ListAssets(ctx context.Context) ([]*models.Asset, error) {
	assets, err := s.vault.GetAssets(ctx, s.ServiceProjectAddress())
	if err != nil {
		return nil, err
	}
	return assets.Items, nil
}

func (s *VaultClient) GetOrCreateAsset(ctx context.Context, name string, ownneraddress string) (*models.Asset, *models.AssetAttribute, error) {
	log := logr.FromContextOrDiscard(ctx)

	assets, err := s.vault.GetAssets(ctx, s.ServiceProjectAddress())
	if err != nil {
		return nil, nil, err
	}
	for _, asset := range assets.Items {
		if len(asset.Attribute) == 0 {
			continue
		}
		attrs, err := models.UnmarshalAssetAttribute(asset.Attribute)
		if err != nil {
			log.Error(err, "decode asset attr")
			continue
		}
		defaultitem, ok := attrs.Items["default"]
		if !ok {
			continue
		}
		if name == defaultitem.Name {
			return asset, attrs, nil
		}
	}
	resp, err := s.vault.MintAsset(ctx,
		s.registerTimeout,
		s.ServiceWallet(),
		s.ServiceProjectAddress(),
		ownneraddress,
	)
	if err != nil {
		return nil, nil, err
	}
	var assetid *big.Int
	if resp.Status == models.TransactionExecPending {
		if err := wait.PollUntilWithContext(ctx, time.Second, func(ctx context.Context) (done bool, err error) {
			resultresp, err := s.vault.GetMintAssetResult(ctx, resp.TxHash)
			if err != nil {
				return false, err
			}
			if resultresp.Status == models.TransactionExecSuccess {
				assetid = big.NewInt(int64(*resultresp.AssetId))
				return true, nil
			}
			return false, nil
		}); err != nil {
			return nil, nil, err
		}
	} else {
		assetid = big.NewInt(int64(*resp.AssetId))
	}
	if err := s.vault.SetAssetAttribute(ctx,
		s.ServiceWallet(),
		nil, // access grant
		s.ServiceProjectAddress(),
		assetid,
		&models.AssetAttributeItem{
			Name:        name,
			Description: name,
			Tags:        []string{"modelx"},
			ImageUri:    "-",
			Attributes:  map[string]interface{}{"key": name},
		},
	); err != nil {
		return nil, nil, err
	}
	asset, err := s.vault.GetAssetById(ctx, s.ServiceProjectAddress(), assetid)
	if err != nil {
		return nil, nil, err
	}
	attrs, err := models.UnmarshalAssetAttribute(asset.Attribute)
	if err != nil {
		log.Error(err, "decode asset attr")
		return nil, nil, err
	}
	return asset, attrs, nil
}

func (s *VaultClient) PutAssetAttr(ctx context.Context, assetid *big.Int, kvs map[string]any) error {
	return s.vault.SetAssetAttribute(ctx,
		s.ServiceWallet(),
		nil,
		s.ServiceProjectAddress(),
		assetid,
		&models.AssetAttributeItem{Attributes: kvs})
}

func (s *VaultClient) PutAssetFile(ctx context.Context, assetid *big.Int, key string, val []byte) error {
	return s.vault.UploadAssetRaw(ctx,
		s.ServiceWallet(),
		nil, // access grant
		s.ServiceProjectAddress(),
		assetid,
		key,
		bytes.NewReader(val))
}

func (s *VaultClient) ListAssetFile(ctx context.Context, assetid *big.Int, prefix string) ([]*models.RawFile, error) {
	list, err := s.vault.ListAssetRaw(ctx, s.ServiceProjectAddress(), assetid)
	if err != nil {
		return nil, err
	}
	retlist := []*models.RawFile{}
	for _, item := range list.Items {
		if prefix != "" || !strings.HasPrefix(item.Name, prefix) {
			continue
		}
		retlist = append(retlist, item)
	}
	return retlist, nil
}

func (s *VaultClient) HasAssetFile(ctx context.Context, assetid *big.Int, key string) (bool, error) {
	// TODO: use vault.HasAssetRaw(key) if possible
	list, err := s.vault.ListAssetRaw(ctx, s.ServiceProjectAddress(), assetid)
	if err != nil {
		return false, err
	}
	for _, item := range list.Items {
		if item.Name == key {
			return true, nil
		}
	}
	return false, nil
}

func (s *VaultClient) GetAssetFile(ctx context.Context, assetid *big.Int, key string) ([]byte, error) {
	val := bytes.NewBuffer(nil)
	if err := s.vault.DownloadAssetRaw(ctx,
		s.ServiceWallet(),
		nil, // access grant
		s.ServiceProjectAddress(),
		assetid,
		key, val,
	); err != nil {
		return nil, err
	}
	return val.Bytes(), nil
}

func (s *VaultClient) RemoveAssetFile(ctx context.Context, assetid *big.Int, key string) error {
	return s.vault.DeleteAssetRaw(ctx,
		s.ServiceWallet(),
		nil, // access grant
		s.ServiceProjectAddress(),
		assetid,
		key,
	)
}

func (s *VaultClient) GenerateAccessGrant(ctx context.Context, assetid *big.Int, allowread, allowwrite bool) (string, error) {
	return s.vault.ShareAssetRaw(ctx,
		s.ServiceWallet(),
		nil, // access grant
		&models.AssetShareList{Assets: []models.AssetList{{
			AssetIds:       []*big.Int{assetid},
			ProjectAddress: s.ServiceProjectAddress(),
		}}},
		sdkvault.AssetShareDisallowReadsOption(!allowread),
		sdkvault.AssetShareDisallowWritesOption(!allowwrite),
	)
}

// GetOrCreateWallet return a wallet for user,if empty return a temp wallet.
func (s *VaultClient) GetOrCreateWallet(ctx context.Context, username string) (*sdkwallet.Wallet, error) {
	log := logr.FromContextOrDiscard(ctx).WithValues("username", username)
	if username == "" {
		mnemonicWords, err := sdkwallet.NewMnemonic()
		if err != nil {
			return nil, err
		}
		return sdkwallet.NewWallet("", mnemonicWords, "")
	}
	val, err := s.kv.Get(ctx, username)
	if err != nil {
		return nil, err
	}
	if val != "" {
		log.Info("exists wallet")
		return sdkwallet.NewWallet(username, val, "")
	}
	existsAccount, err := s.vault.GetAccountByUsername(ctx, username)
	if err != nil {
		if !errors.Is(err, sdkerrs.ErrorAccountNotFound) {
			return nil, err
		}
		// create useraccount
		resp, wallet, mnemonicWords, err := s.vault.RegisterAccount(ctx, s.registerTimeout, username, "")
		if err != nil {
			return nil, err
		}
		log.Info("created wallet", "resp", resp)
		if err := s.kv.Set(ctx, username, mnemonicWords); err != nil {
			return nil, err
		}
		return wallet, nil
	}
	_ = existsAccount
	// privkey ?
	return nil, fmt.Errorf("unable to recovery account: %s", existsAccount.Username)
}

// ServiceWallet return modelx's wallet
func (s *VaultClient) ServiceWallet() *sdkwallet.Wallet {
	return s.serviceWallet
}

type KVDB interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, val string) error
}

type LocalKVDB struct {
	db     *leveldb.DB
	bucket []byte
}

func NewLocalKVStore(path string) (*LocalKVDB, error) {
	if path == "" {
		return nil, fmt.Errorf("local store path not set")
	}
	if basepath := filepath.Dir(path); basepath != "" {
		os.MkdirAll(basepath, os.ModePerm)
	}
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}
	return &LocalKVDB{db: db}, nil
}

func (s *LocalKVDB) Get(ctx context.Context, key string) (string, error) {
	val, err := s.db.Get([]byte(key), nil)
	if err != nil {
		// ignore not found error
		if errors.Is(err, leveldb.ErrNotFound) {
			return "", nil
		}
		return "", err
	}
	return string(val), nil
}

func (s *LocalKVDB) Set(ctx context.Context, key string, val string) error {
	return s.db.Put([]byte(key), []byte(val), nil)
}
