package client

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/opencontainers/go-digest"
	"golang.org/x/exp/slices"
	"kubegems.io/modelx/pkg/types"
	"sigs.k8s.io/yaml"
)

const ModelConfigFileName = "modelx.yaml"

type Model struct {
	BaseDir  string
	Manifest types.Manifest
}

func PackLocalModel(ctx context.Context, dir string) (*Model, error) {
	manifest := types.Manifest{
		Blobs: []types.Descriptor{},
	}
	fsys := os.DirFS(dir)
	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		f, err := fsys.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		desc := types.Descriptor{
			Name: path,
		}
		if fi, err := os.Stat(path); err == nil {
			desc.Size = fi.Size()
		}
		switch path {
		case ModelConfigFileName:
			configcontent, err := io.ReadAll(f)
			if err != nil {
				return err
			}
			var modelconfig types.ModelConfig
			if err := yaml.Unmarshal(configcontent, &modelconfig); err != nil {
				return fmt.Errorf("parse model config:%s %w", ModelConfigFileName, err)
			}

			manifest.Annotations = modelconfig.Annotations
			if manifest.Annotations == nil {
				manifest.Annotations = map[string]string{}
			}
			manifest.Annotations[types.AnnotationDescription] = modelconfig.Description
			desc.Digest = digest.FromBytes(configcontent)
			desc.MediaType = types.MediaTypeModelConfigYaml
		default:
			digest, err := digest.FromReader(f)
			if err != nil {
				return err
			}
			desc.Digest = digest
			desc.MediaType = types.MediaTypeModelFile
		}

		manifest.Blobs = append(manifest.Blobs, desc)
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.SortFunc(manifest.Blobs, types.SortDescriptorName)
	return &Model{Manifest: manifest, BaseDir: dir}, nil
}
