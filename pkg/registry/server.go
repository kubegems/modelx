package registry

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/coreos/go-oidc/v3/oidc"
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
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(opt.S3.AccessKey, opt.S3.SecretKey, ""),
		),
		config.WithRegion(opt.S3.Region),
		config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(
				func(service, region string, options ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{URL: opt.S3.URL}, nil
				},
			),
		),
	)
	if err != nil {
		return nil, err
	}
	s3cli := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})
	storage := &S3StorageProvider{
		Bucket:  opt.S3.Buket,
		Client:  s3cli,
		Expire:  opt.S3.PresignExpire,
		Prefix:  "registry",
		PreSign: s3.NewPresignClient(s3cli),
	}
	store := &RegistryStore{
		Storage:        storage,
		EnableRedirect: opt.EnableRedirect,
	}
	return &Registry{Manifest: store}, nil
}

func NewOIDCAuthFilter(ctx context.Context, issuer string, next http.Handler) http.Handler {
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		log.Fatal(err)
	}
	verifier := provider.Verifier(&oidc.Config{SkipClientIDCheck: true})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headerAuthorzation := r.Header.Get("Authorization")
		token := strings.TrimPrefix(headerAuthorzation, "Bearer ")
		if token == "" {
			token = r.URL.Query().Get("access_token")
		}
		if len(token) == 0 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		idtoken, err := verifier.Verify(r.Context(), token)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		r.Header.Set("username", idtoken.Subject)
		next.ServeHTTP(w, r)
	})
}
