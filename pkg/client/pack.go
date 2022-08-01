package client

import (
	"context"
	"io/fs"
	"os"

	"github.com/opencontainers/go-digest"
	"golang.org/x/exp/slices"
	"kubegems.io/modelx/pkg/types"
)

type Package struct {
	types.Manifest
	BaseDir string
}

func PackManifest(ctx context.Context, dir string, configfile string, annotations map[string]string) (*Package, error) {
	manifest := types.Manifest{
		Blobs:       []types.Descriptor{},
		Annotations: annotations,
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

		fi, err := fs.Stat(fsys, path)
		if err != nil {
			return err
		}

		digest, err := digest.FromReader(f)
		if err != nil {
			return err
		}

		desc := types.Descriptor{
			Name:     path,
			Digest:   digest,
			Size:     fi.Size(),
			Modified: fi.ModTime(),
		}

		if path == configfile {
			manifest.Config = desc
		} else {
			manifest.Blobs = append(manifest.Blobs, desc)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.SortFunc(manifest.Blobs, types.SortDescriptorName)
	return &Package{Manifest: manifest, BaseDir: dir}, nil
}
