package registry

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/go-logr/logr"
)

func Run(ctx context.Context, opts *Options) error {
	log := logr.FromContextOrDiscard(ctx)
	registry, err := NewRegistry(ctx, opts)
	if err != nil {
		return err
	}

	handler := registry.route()
	handler = LoggingFilter(log, handler)

	if opts.OIDC.Issuer != "" {
		handler = NewOIDCAuthFilter(ctx, opts.OIDC.Issuer, handler)
	}

	server := http.Server{
		Addr:    opts.Listen,
		Handler: handler,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}
	go func() {
		<-ctx.Done()
		server.Shutdown(ctx)
	}()
	if opts.TLS.CertFile != "" && opts.TLS.KeyFile != "" {
		log.Info("registry listening", "https", opts.Listen)
		return server.ListenAndServeTLS(opts.TLS.CertFile, opts.TLS.KeyFile)
	} else {
		log.Info("registry listening", "http", opts.Listen)
		return server.ListenAndServe()
	}
}

func NewRegistry(ctx context.Context, opt *Options) (*Registry, error) {
	log := logr.FromContextOrDiscard(ctx)
	log.Info("prepare registry", "options", opt)
	var registryStore RegistryStore
	if opt.Vault.Address != "" {
		vaultRegistrystore, err := NewVaultRegistryStore(ctx, opt.Vault)
		if err != nil {
			return nil, err
		}
		registryStore = vaultRegistrystore
	} else if opt.S3.URL != "" {
		fsstore, err := NewFSRegistryStore(ctx, opt.S3, opt.EnableRedirect)
		if err != nil {
			return nil, err
		}
		registryStore = fsstore
	} else {
		return nil, fmt.Errorf("no storage backend set")
	}
	return &Registry{Store: registryStore}, nil
}
