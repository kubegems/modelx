package vault

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/aws/smithy-go/transport/http"
	"github.com/go-logr/logr"
	"github.com/syndtr/goleveldb/leveldb"
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

	// cache
	assetsCache *AssetCache
}

func NewVaultClient(ctx context.Context, options *VaultOptions) (*VaultClient, error) {
	vault, err := sdkvault.NewVault(ctx, options.Address, nil)
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
	cli := &VaultClient{
		kv:              kv,
		vault:           vault,
		registerTimeout: DefaultRegisterTimeout,
		serviceWallet:   serviceWallet,
		serviceProject:  serviceProject,
		vaultServiceUrl: options.Address,
	}
	cli.assetsCache = NewAssetCache(cli)
	if err := cli.assetsCache.InitSync(ctx); err != nil {
		return nil, err
	}
	return cli, nil
}

func initProject(ctx context.Context, vault *sdkvault.Vault, wallet *sdkwallet.Wallet, options *VaultOptions) (*models.Project, error) {
	log := logr.FromContextOrDiscard(ctx).WithValues("project", options.Project)
	log.Info("init service project")
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
		err := retry.Do(func() error {
			resp, err := vault.GetCreateGeneralProjectResult(ctx, resp.TxHash)
			if err != nil {
				return err
			}
			if resp.Status == models.TransactionExecPending {
				log.Info("waiting project transcation")
				return ErrTxPending
			}
			projectaddress = *resp.ProjectAddress
			return nil
		}, retry.Context(ctx))
		if err != nil {
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

func (s *VaultClient) GenerateBolbURL(ctx context.Context, username string, assetid *big.Int, filename string, allowread, allowwrite bool) (string, error) {
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
		return "", fmt.Errorf("share asset:%w", err)
	}
	vaultassetmeta := VaultBlobMeta{
		ServiceUrl:     s.vaultServiceUrl,
		ProjectAddress: s.ServiceProjectAddress(),
		AssetID:        assetid,
		AccessGrant:    accessgrant,
		Username:       username,
		File:           filename,
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

func (s *VaultClient) GetAsset(ctx context.Context, name string) (*models.Asset, error) {
	return s.assetsCache.GetByName(ctx, name, "")
}

func (s *VaultClient) GetOrCreateAsset(ctx context.Context, name string, ownneraddress string) (*models.Asset, error) {
	return s.assetsCache.GetByName(ctx, name, ownneraddress)
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
	return s.PutAssetFileFromReader(ctx, assetid, key, bytes.NewReader(val))
}

func (s *VaultClient) PutAssetFileFromReader(ctx context.Context, assetid *big.Int, key string, r io.ReadSeeker) error {
	return s.vault.UploadAssetRaw(ctx,
		s.ServiceWallet(),
		nil, // access grant
		s.ServiceProjectAddress(),
		assetid,
		key,
		r)
}

func (s *VaultClient) ListAssetFile(ctx context.Context, assetid *big.Int, prefix string) ([]*models.RawFile, error) {
	list, err := s.vault.ListAssetRaw(ctx, s.ServiceProjectAddress(), assetid)
	if err != nil {
		return nil, err
	}
	retlist := []*models.RawFile{}
	for _, item := range list.Items {
		if prefix != "" && !strings.HasPrefix(item.Name, prefix) {
			continue
		}
		retlist = append(retlist, item)
	}
	return retlist, nil
}

func (s *VaultClient) HasAssetFile(ctx context.Context, assetid *big.Int, key string) (bool, error) {
	return s.vault.AssetHasRaw(ctx, s.ServiceProjectAddress(), assetid, key)
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
	if err := s.vault.DeleteAssetRaw(ctx,
		s.ServiceWallet(),
		nil, // access grant
		s.ServiceProjectAddress(),
		assetid,
		key,
	); err != nil {
		if IsStorageNotFound(err) {
			return nil
		}
		return err
	}
	return nil
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
			return nil, fmt.Errorf("get account: %w", err)
		}
		// create useraccount
		resp, wallet, mnemonicWords, err := s.vault.RegisterAccount(ctx, s.registerTimeout, username, "")
		if err != nil {
			return nil, fmt.Errorf("register account: %w", err)
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

func IsStorageNotFound(err error) bool {
	var apie *http.ResponseError
	if errors.As(err, &apie) {
		return apie.HTTPStatusCode() == 404
	}
	return false
}
