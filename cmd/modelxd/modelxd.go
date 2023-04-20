package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
	"github.com/spf13/cobra"
	"kubegems.io/modelx/pkg/registry"
	"kubegems.io/modelx/pkg/version"
)

const ErrExitCode = 1

func main() {
	if err := NewRegistryCmd().Execute(); err != nil {
		fmt.Println(err.Error())
		os.Exit(ErrExitCode)
	}
}

func NewRegistryCmd() *cobra.Command {
	options := registry.DefaultOptions()
	cmd := &cobra.Command{
		Use:     "modelxd",
		Short:   "modelxd",
		Version: version.Get().String(),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
			defer cancel()

			log.SetFlags(log.LstdFlags | log.Lshortfile)
			ctx = logr.NewContext(ctx, stdr.NewWithOptions(log.Default(), stdr.Options{LogCaller: stdr.Error}))

			return registry.Run(ctx, options)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&options.Listen, "listen", options.Listen, "listen address")
	flags.StringVar(&options.TLS.CAFile, "tls-ca", options.TLS.CAFile, "tls ca file")
	flags.StringVar(&options.TLS.CertFile, "tls-cert", options.TLS.CertFile, "tls cert file")
	flags.StringVar(&options.TLS.KeyFile, "tls-key", options.TLS.KeyFile, "tls key file")
	flags.StringVar(&options.S3.Buket, "s3-bucket", options.S3.Buket, "s3 bucket")
	flags.StringVar(&options.S3.URL, "s3-url", options.S3.URL, "s3 url")
	flags.StringVar(&options.S3.AccessKey, "s3-access-key", options.S3.AccessKey, "s3 access key")
	flags.StringVar(&options.S3.SecretKey, "s3-secret-key", options.S3.SecretKey, "s3 secret key")
	flags.DurationVar(&options.S3.PresignExpire, "s3-presign-expire", options.S3.PresignExpire, "s3 presign expire")
	flags.StringVar(&options.S3.Region, "s3-region", options.S3.Region, "s3 region")
	flags.StringVar(&options.OIDC.Issuer, "oidc-issuer", options.OIDC.Issuer, "oidc issuer")
	flags.BoolVar(&options.EnableRedirect, "enable-redirect", options.EnableRedirect, "enable blob storage redirect")

	return cmd
}
