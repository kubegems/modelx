package model

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	"kubegems.io/modelx/cmd/modelx/repo"
)

const (
	AnnotationDescription = "modelx.model.description"
	AnnotationMintainers  = "modelx.model.maintainers"
)
const ModelConfigFileName = "modelx.yaml"

type ModelConfig struct {
	Description string            `json:"description"`
	FrameWork   string            `json:"framework"`
	Tags        []string          `json:"tags"`
	Mantainers  []string          `json:"maintainers"`
	Annotations map[string]string `json:"annotations,omitempty"`
	ModelFiles  []string          `json:"modelFiles"`
	Config      any               `json:"config"`
}

func NewInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "get config of model",
		Example: `
  modex info  https://registry.example.com/repo/name@version
		`,
		SilenceUsage: true,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return repo.CompleteRegistryRepositoryVersion(toComplete)
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
			defer cancel()
			if len(args) == 0 {
				return errors.New("at least one argument is required")
			}
			config, err := GetConfig(ctx, args[0])
			if err != nil {
				return err
			}
			fmt.Print(string(config))
			return nil
		},
	}
	return cmd
}

func GetConfig(ctx context.Context, ref string) ([]byte, error) {
	reference, err := ParseReference(ref)
	if err != nil {
		return nil, err
	}
	if reference.Repository == "" {
		return nil, errors.New("repository is not specified")
	}
	cli := reference.Client()
	manfiest, err := cli.GetManifest(ctx, reference.Repository, reference.Version)
	if err != nil {
		return nil, err
	}
	if content, _, err := cli.Remote.GetBlob(ctx, reference.Repository, manfiest.Config.Digest); err != nil {
		return nil, err
	} else {
		defer content.Close()
		return io.ReadAll(content)
	}
}
