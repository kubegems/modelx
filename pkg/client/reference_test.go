package client

import (
	"reflect"
	"testing"
)

func TestParseReference(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    Reference
		wantErr bool
	}{
		{
			name: "valid",
			raw:  "https://registry.example.com/repository@sha256:abcdef",
			want: Reference{
				Registry:   "https://registry.example.com",
				Repository: "repository",
				Version:    "sha256:abcdef",
			},
		},
		{
			raw: "https://registry.example.com:8443/repository/name@v1",
			want: Reference{
				Registry:   "https://registry.example.com:8443",
				Repository: "repository/name",
				Version:    "v1",
			},
		},
		{
			raw: "https://registry.example.com/repo/name",
			want: Reference{
				Registry:   "https://registry.example.com",
				Repository: "repository/name",
				Version:    "latest",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseReference(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseReference() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseReference() = %v, want %v", got, tt.want)
			}
		})
	}
}
