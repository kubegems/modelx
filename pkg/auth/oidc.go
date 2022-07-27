package auth

import (
	"context"

	"github.com/coreos/go-oidc/v3/oidc"
)

func NewOIDCProvider(ctx context.Context, issuer string) (*oidc.Provider, error) {
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, err
	}
	return provider, nil
}
