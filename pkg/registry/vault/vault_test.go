package vault


import (
	"context"
	"testing"
)

func TestNewVaultStore(t *testing.T) {
	ctx := context.Background()
	options := NewDefaultVaultOptions()

	options.Address = "https://example.com"
	options.Username = "modelx"
	options.Mnemonic = "some word"

	cli, err := NewVaultClient(ctx, options)
	if err != nil {
		t.Error(err)
		return
	}

	asset, err := cli.GetOrCreateAsset(ctx, "demo-repositoey/name", cli.serviceWallet.GetAddress())
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(asset)

	assets, err := cli.ListAssets(ctx)
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(assets)

	if err := cli.PutAssetFile(ctx, asset.AssetId, "test-key", []byte("content")); err != nil {
		t.Error(err)
		return
	}
	files, err := cli.ListAssetFile(ctx, asset.AssetId, "")
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(files)

	wallet := cli.ServiceWallet()
	if wallet.GetUserName() != options.Username {
		t.Errorf("service wallet username %s != %s", wallet.GetUserName(), options.Username)
		return
	}

	testusername := "test1"
	userwallet, err := cli.GetOrCreateWallet(ctx, testusername)
	if err != nil {
		t.Error(err)
		return
	}
	if userwallet.GetUserName() != testusername {
		t.Errorf("testuser wallet username %s != %s", wallet.GetUserName(), options.Username)
		return
	}

	tempwallet, err := cli.GetOrCreateWallet(ctx, "")
	if err != nil {
		t.Error(err)
		return
	}
	address := tempwallet.GetAddress()
	t.Logf("temproary wallet address: %s", address)
}
