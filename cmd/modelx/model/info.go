package model

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	"kubegems.io/modelx/pkg/client"
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
	Config      any               `json:"config"`
}

func NewInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "info <url>",
		Example: `
  modex info  https://registry.example.com/repo/name@version
		`,
		SilenceUsage: true,
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
	manfiest, err := client.GetManifest(ctx, reference)
	if err != nil {
		return nil, err
	}
	content := bytes.NewBuffer(nil)
	if err := client.PullBlob(ctx, reference, content, manfiest.Config, nil); err != nil {
		return nil, err
	}
	return content.Bytes(), nil
}
