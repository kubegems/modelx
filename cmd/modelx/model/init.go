package model

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

func NewInitCmd() *cobra.Command {
	force := false
	cmd := &cobra.Command{
		Use:   "init",
		Short: "init an new model at path",
		Example: `
  modex init .
		`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
			defer cancel()
			if len(args) == 0 {
				return errors.New("at least one argument is required")
			}
			if err := InitModelx(ctx, args[0], force); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "force init")
	return cmd
}

func InitModelx(ctx context.Context, path string, force bool) error {
	if _, err := os.Stat(path); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else {
		if !force {
			return fmt.Errorf("path %s already exists, remove it first", path)
		}
	}

	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("create modelx directory:%s %w", path, err)
	}
	config := ModelConfig{
		Description: "This is a modelx model",
		FrameWork:   "<some framework>",
		Config: map[string]interface{}{
			"inputs":  map[string]interface{}{},
			"outputs": map[string]interface{}{},
		},
		Tags: []string{
			"modelx",
			"<other>",
		},
		Mantainers: []string{
			"maintainer",
		},
		ModelFiles: []string{},
	}
	configcontent, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("encode model %w", err)
	}
	configfile := filepath.Join(path, ModelConfigFileName)
	if err := os.WriteFile(configfile, configcontent, 0o644); err != nil {
		return fmt.Errorf("write model config:%s %w", configfile, err)
	}

	fmt.Printf("Modelx model initialized in %s\n", path)
	return nil
}
