package model

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"kubegems.io/modelx/cmd/modelx/repo"
	"kubegems.io/modelx/pkg/client"
)

func NewPushCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push",
		Short: "push a model to a modelx repository",
		Example: `
  modex push modelx/hello/gpt@v1 .
		`,
		SilenceUsage: true,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return repo.CompleteRegistryRepositoryVersion(toComplete)
			}
			if len(args) == 1 {
				return nil, cobra.ShellCompDirectiveFilterDirs
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
			defer cancel()
			if len(args) == 0 {
				return errors.New("at least one argument is required")
			}
			if len(args) == 1 {
				args = append(args, "")
			}
			if err := PushModel(ctx, args[0], args[1]); err != nil {
				return err
			}
			return nil
		},
	}
	return cmd
}

func PushModel(ctx context.Context, ref string, dir string) error {
	reference, err := ParseReference(ref)
	if err != nil {
		return err
	}
	if dir == "" {
		dir = "."
	}
	// parse annotations from model config
	configcontent, err := os.ReadFile(filepath.Join(dir, ModelConfigFileName))
	if err != nil {
		return fmt.Errorf("read model config:%s %w", ModelConfigFileName, err)
	}
	var config ModelConfig
	if err := yaml.Unmarshal(configcontent, &config); err != nil {
		return fmt.Errorf("parse model config:%s %w", ModelConfigFileName, err)
	}

	annotations := map[string]string{}
	for k, v := range config.Annotations {
		annotations[k] = v
	}
	annotations[AnnotationDescription] = config.Description

	// manifest
	manifest, err := client.PackManifest(ctx, dir, ModelConfigFileName, annotations)
	if err != nil {
		return err
	}
	fmt.Printf("Pushing to %s \n", reference.String())
	return reference.Client().Push(ctx, reference.Repository, reference.Version, manifest, dir)
}
