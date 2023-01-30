package types

import (
	"math/big"
	"reflect"
	"testing"
)

func TestVaultBlobMeta_ToURL(t *testing.T) {
	type fields struct {
		ServiceHost    string
		ProjectAddress string
		AccessGrant    string
		AssetID        *big.Int
		Username       string
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				ServiceHost:    "example.com",
				ProjectAddress: "0x1234567890abcdef",
				AccessGrant:    "1234567890",
				AssetID:        big.NewInt(1),
				Username:       "anonymous",
			},
			want: "idoe://example.com/0x1234567890abcdef/1?access-grant=1234567890&username=anonymous",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := VaultBlobMeta{
				ServiceUrl:     tt.fields.ServiceHost,
				ProjectAddress: tt.fields.ProjectAddress,
				AccessGrant:    tt.fields.AccessGrant,
				AssetID:        tt.fields.AssetID,
				Username:       tt.fields.Username,
			}
			got, err := u.ToURL()
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
				in: "idoe://example.com/0x1234567890abcdef/1?access-grant=1234567890&username=anonymous",
			},
			want: &VaultBlobMeta{
				ServiceUrl:     "example.com",
				ProjectAddress: "0x1234567890abcdef",
				AccessGrant:    "1234567890",
				AssetID:        big.NewInt(1),
				Username:       "anonymous",
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
