package registry

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
)

func Run(ctx context.Context, opts *Options) error {
	registry, err := NewRegistry(ctx, opts)
	if err != nil {
		return err
	}

	handler := registry.route()
	handler = handlers.CombinedLoggingHandler(os.Stdout, handler)

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
		log.Printf("registry listening on https: %s", opts.Listen)
		return server.ListenAndServeTLS(opts.TLS.CertFile, opts.TLS.KeyFile)
	} else {
		log.Printf("registry listening on http %s", opts.Listen)
		return server.ListenAndServe()
	}
}

func NewRegistry(ctx context.Context, opt *Options) (*Registry, error) {
	var registryStore RegistryStore
	if opt.Vault.Address != "" {
		vaultRegistrystore, err := NewVaultRegistryStore(ctx, opt.Vault)
		if err != nil {
			return nil, err
		}
		registryStore = vaultRegistrystore
	}
	if opt.S3.URL != "" {
		fsstore, err := NewFSRegistryStore(ctx, opt.S3, opt.EnableRedirect)
		if err != nil {
			return nil, err
		}
		registryStore = fsstore
	}
	if registryStore == nil {
		return nil, fmt.Errorf("no storage backend set")
	}
	return &Registry{Store: registryStore}, nil
}
