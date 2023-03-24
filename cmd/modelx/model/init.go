package model

import (
	"context"
	"errors"
	"fmt"
	"os"
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
			ctx, cancel := BaseContext()
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
			return fmt.Errorf("path %s already exists", path)
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
		Resources: map[string]any{
			"cpu":    "4",
			"memory": "16Gi",
			"gpu": map[string]any{
				"nvdia": map[string]any{
					"nvdia/gpu": "1",
				},
				"gpu-manager": map[string]any{
					"tencent.com/vcuda-core":   "50",
					"tencent.com/vcuda-memory": "25",
				},
			},
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
	if err := os.WriteFile(configfile, configcontent, 0o755); err != nil {
		return fmt.Errorf("write model config:%s %w", configfile, err)
	}

	// Init README.md
	basefile := filepath.Base(path)
	if basefile != "" {
		readmefile := filepath.Join(path, ReadmeFileName)
		_, err := os.Stat(readmefile)
		if errors.Is(err, os.ErrNotExist) {
			readmecontent := fmt.Sprintf("# %s\n\nAwesome model descrition.\n", basefile)
			os.WriteFile(readmefile, []byte(readmecontent), 0o755)
		}
	}

	fmt.Printf("Modelx model initialized in %s\n", path)
	return nil
}
