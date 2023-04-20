package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"sync"

	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"kubegems.io/modelx/pkg/registry/vault"
	sdkaccess "src.cloudminds.com/blockchain/vault/vault-sdk-go/sdk/access"
	sdkvault "src.cloudminds.com/blockchain/vault/vault-sdk-go/sdk/vault"
	sdkwallet "src.cloudminds.com/blockchain/vault/vault-sdk-go/sdk/wallet"
)

func init() {
	GlobalExtensions["idoe"] = &IdoeExt{}
	GlobalExtensions["idoes"] = &IdoeExt{}
}

type IdoeExt struct {
	vaultcache     map[string]*sdkvault.Vault
	tempraryWallet *sdkwallet.Wallet
	mu             sync.Mutex
}

func (e *IdoeExt) tempWallet() (*sdkwallet.Wallet, error) {
	if e.tempraryWallet != nil {
		return e.tempraryWallet, nil
	}
	words, err := sdkwallet.NewMnemonic()
	if err != nil {
		return nil, err
	}
	wallet, err := sdkwallet.NewWallet("", words, "")
	if err != nil {
		return nil, err
	}
	e.tempraryWallet = wallet
	return wallet, nil
}

func (e *IdoeExt) vaultOf(ctx context.Context, serviceaddress string) (*sdkvault.Vault, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.vaultcache == nil {
		e.vaultcache = map[string]*sdkvault.Vault{}
	}
	if vault, ok := e.vaultcache[serviceaddress]; ok {
		return vault, nil
	}
	vault, err := sdkvault.NewVault(ctx, serviceaddress, nil)
	if err != nil {
		return nil, err
	}
	return vault, nil
}

func (e *IdoeExt) Download(ctx context.Context, location *url.URL, into io.Writer) error {
	assetmeta, err := vault.ParseVaultURL(location.String())
	if err != nil {
		return err
	}
	grant, err := sdkaccess.ParseAccessGrant(assetmeta.AccessGrant)
	if err != nil {
		return err
	}
	wallet, err := e.tempWallet()
	if err != nil {
		return err
	}
	vault, err := e.vaultOf(ctx, assetmeta.ServiceUrl)
	if err != nil {
		return err
	}
	if intoat, ok := into.(io.WriterAt); !ok {
		buffer, ok := into.(*bytes.Buffer)
		if !ok {
			return fmt.Errorf("%V neither io.WriterAt nor bytes.Buffer", into)
		}
		buf := manager.NewWriteAtBuffer(nil)
		if err := vault.DownloadAssetRawForServerSideEncryption(ctx,
			wallet,
			grant,
			assetmeta.ProjectAddress,
			assetmeta.AssetID,
			assetmeta.File,
			buf); err != nil {
			return err
		}
		_, err := buffer.Write(buf.Bytes())
		return err
	} else {
		return vault.DownloadAssetRawForServerSideEncryption(ctx, wallet, grant, assetmeta.ProjectAddress, assetmeta.AssetID, assetmeta.File, intoat)
	}
}

func (e *IdoeExt) Upload(ctx context.Context, location *url.URL, blob DescriptorWithContent) error {
	assetmeta, err := vault.ParseVaultURL(location.String())
	if err != nil {
		return err
	}
	grant, err := sdkaccess.ParseAccessGrant(assetmeta.AccessGrant)
	if err != nil {
		return err
	}
	wallet, err := e.tempWallet()
	if err != nil {
		return err
	}
	vault, err := e.vaultOf(ctx, assetmeta.ServiceUrl)
	if err != nil {
		return err
	}
	content, err := blob.GetContent()
	if err != nil {
		return err
	}
	return vault.UploadAssetRawWithHash(ctx,
		wallet,
		grant,
		assetmeta.ProjectAddress,
		assetmeta.AssetID,
		assetmeta.File,
		content,
		[]byte(blob.Digest.Hex()),
	)
}
