package model

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"path"

	"github.com/spf13/cobra"
	"kubegems.io/modelx/pkg/client"
)

func NewPullCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "pull <url> <dir>",
		Example: `
  modex pull  https://registry.example.com/repo/name@version .
		`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
			defer cancel()
			if len(args) == 0 {
				return errors.New("at least one argument is required")
			}
			if len(args) == 1 {
				args = append(args, "")
			}
			return PullModelx(ctx, args[0], args[1])
		},
	}
	return cmd
}

func PullModelx(ctx context.Context, ref string, into string) error {
	reference, err := ParseReference(ref)
	if err != nil {
		return err
	}
	if reference.Repository == "" {
		return errors.New("repository is not specified")
	}
	if into == "" {
		into = path.Base(reference.Repository)
	}
	manifest, err := client.GetManifest(ctx, reference)
	if err != nil {
		return err
	}
	pack := client.Package{Manifest: *manifest, BaseDir: into}
	return client.PullPack(ctx, reference, pack)
}
