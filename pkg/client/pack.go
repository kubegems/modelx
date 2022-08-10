package client

import (
	"context"
	"io/fs"
	"os"
	"strings"

	"golang.org/x/exp/slices"
	"kubegems.io/modelx/pkg/types"
)

type Package struct {
	types.Manifest
	BaseDir string
}

func PackManifest(ctx context.Context, dir string, configfile string, annotations map[string]string) (types.Manifest, error) {
	manifest := types.Manifest{
		Blobs:       []types.Descriptor{},
		Annotations: annotations,
	}
	fsys := os.DirFS(dir)
	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.HasPrefix(path, ".") {
			return nil
		}
		if d.IsDir() {
			if strings.HasPrefix(path, ".") {
				return fs.SkipDir
			}
			return nil
		}
		desc := types.Descriptor{Name: path}
		if path == configfile {
			manifest.Config = desc
		} else {
			manifest.Blobs = append(manifest.Blobs, desc)
		}
		return nil
	})
	if err != nil {
		return types.Manifest{}, err
	}
	slices.SortFunc(manifest.Blobs, types.SortDescriptorName)
	return manifest, nil
}
