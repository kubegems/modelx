package types

import (
	"math/big"
	"reflect"
	"testing"
)

func TestVaultBlobMeta_ToURL(t *testing.T) {
	tests := []struct {
		name    string
		fields  VaultBlobMeta
		want    string
		wantErr bool
	}{
		{
			name: "default",
			fields: VaultBlobMeta{
				ServiceUrl:     "example.com",
				ProjectAddress: "0x1234567890abcdef",
				AccessGrant:    "1234567890",
				AssetID:        big.NewInt(1),
				Username:       "anonymous",
				File:           "blob/sha256:123456",
			},
			want: "idoe://example.com?access-grant=1234567890&asset-id=1&file=blob%2Fsha256%3A123456&project-address=0x1234567890abcdef&username=anonymous",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.fields.ToURL()
			if (err != nil) != tt.wantErr {
				t.Errorf("VaultBlobMeta.ToURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("VaultBlobMeta.ToURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseVaultURL(t *testing.T) {
	type args struct {
		in string
	}
	tests := []struct {
		name    string
		args    args
		want    *VaultBlobMeta
		wantErr bool
	}{
		{
			name: "default",
			args: args{
				in: "idoes://example.com?access-grant=1234567890&asset-id=1&file=blob%2Fsha256%3A123456&project-address=0x1234567890abcdef&username=anonymous",
			},
			want: &VaultBlobMeta{
				ServiceUrl:     "https://example.com",
				ProjectAddress: "0x1234567890abcdef",
				AccessGrant:    "1234567890",
				AssetID:        big.NewInt(1),
				Username:       "anonymous",
				File:           "blob/sha256:123456",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseVaultURL(tt.args.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVaultURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseVaultURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
