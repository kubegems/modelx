package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/opencontainers/go-digest"
	"kubegems.io/modelx/pkg/client/progress"
	"kubegems.io/modelx/pkg/types"
)

func (c Client) Pull(ctx context.Context, repo string, version string, into string) error {
	// check if the directory exists and is empty
	if dirInfo, err := os.Stat(into); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err := os.MkdirAll(into, 0o755); err != nil {
			return fmt.Errorf("create directory %s: %v", into, err)
		}
	} else {
		if !dirInfo.IsDir() {
			return fmt.Errorf("%s is not a directory", into)
		}
	}

	manifest, err := c.GetManifest(ctx, repo, version)
	if err != nil {
		return err
	}
	return c.PullBlobs(ctx, repo, into, append(manifest.Blobs, manifest.Config), false)
}

func (c Client) PullBlobs(ctx context.Context, repo string, basedir string, blobs []types.Descriptor, usecache bool) error {
	mb := progress.NewMuiltiBar(os.Stdout, 40)
	go mb.Run(ctx)

	for _, blob := range blobs {
		blob := blob
		mb.Go(blob.Name, "pending", func(b *progress.Bar) error {
			return c.PullBlob(ctx, repo, blob, basedir, b, usecache)
		})
	}
	return mb.Wait()
}

func checkLocalBlob(ctx context.Context, dir string, desc types.Descriptor) (bool, error) {
	localfilename := filepath.Join(dir, desc.Name)

	fi, err := os.Stat(localfilename)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if fi.IsDir() {
		digest, err := TGZ(ctx, localfilename, "")
		if err != nil {
			return false, err
		}
		if digest.String() == desc.Digest.String() {
			return true, nil
		}
		return false, nil
	}

	// file exists, check hash
	if f, err := os.Open(localfilename); err == nil {
		defer f.Close()
		digest, err := digest.FromReader(f)
		if err != nil {
			return false, err
		}
		if digest.String() == desc.Digest.String() {
			return true, nil
		}
	}
	return false, nil
}

func writeFile(filename string, src io.ReadCloser, perm os.FileMode) error {
	var f *os.File
	var err error

	if perm == 0 {
		perm = 0o644
	}

	f, err = os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm.Perm())
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(filename), os.ModePerm); err != nil {
			return err
		}
		f, err = os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm.Perm())
		if err != nil {
			return err
		}
	}

	defer f.Close()
	defer src.Close()

	_, err = io.Copy(f, src)
	return err
}

func (c Client) PullBlob(ctx context.Context, repo string, desc types.Descriptor, basedir string, bar *progress.Bar, usecache bool) error {
	switch desc.MediaType {
	case MediaTypeModelDirectoryTarGz:
		return c.pullDirectory(ctx, repo, desc, basedir, bar, usecache)
	case MediaTypeModelFile:
		return c.pullFile(ctx, repo, desc, basedir, bar)
	case MediaTypeModelConfigYaml:
		return c.pullConfig(ctx, repo, desc, basedir, bar)
	default:
		return fmt.Errorf("unsupported media type %s", desc.MediaType)
	}
}

func (c Client) pullConfig(ctx context.Context, repo string, desc types.Descriptor, basedir string, bar *progress.Bar) error {
	return c.pullFile(ctx, repo, desc, basedir, bar)
}

func (c Client) pullFile(ctx context.Context, repo string, desc types.Descriptor, basedir string, bar *progress.Bar) error {
	// check hash
	bar.SetStatus(desc.Name, "checking")
	filename := filepath.Join(basedir, desc.Name)
	if f, err := os.Open(filename); err == nil {
		digest, err := digest.FromReader(f)
		if err != nil {
			return err
		}
		if digest.String() == desc.Digest.String() {
			bar.SetProgress(desc.Size, desc.Size)
			bar.SetStatus(desc.Digest.Hex()[:8], "already exists")
			return nil
		}
		_ = f.Close()
	} else if !os.IsNotExist(err) {
		return err
	}

	var content io.ReadCloser
	var contentlen int64
	if desc.Digest == EmptyFileDigiest {
		content, contentlen = io.NopCloser(bytes.NewReader(nil)), 0
	} else {
		// pull
		ctt, cttl, err := c.Remote.GetBlob(ctx, repo, desc.Digest)
		if err != nil {
			return err
		}
		content, contentlen = ctt, cttl
	}
	content = bar.WrapReader(content, desc.Digest.Hex()[:8], contentlen, "downloading", "done", "failed")
	return writeFile(filename, content, desc.Mode.Perm())
}

func (c Client) pullDirectory(ctx context.Context, repo string, desc types.Descriptor, basedir string, bar *progress.Bar, useCache bool) error {
	// check hash
	bar.SetStatus(desc.Name, "checking")
	digest, err := TGZ(ctx, filepath.Join(basedir, desc.Name), "")
	if err != nil {
		return err
	}
	if digest.String() == desc.Digest.String() {
		bar.SetStatus(desc.Digest.Hex()[:8], "already exists")
		return nil
	}

	// pull to cache
	content, contentlen, err := c.Remote.GetBlob(ctx, repo, desc.Digest)
	if err != nil {
		return err
	}
	src := bar.WrapReader(content, desc.Digest.Hex()[:8], contentlen, "downloading", "done", "failed")
	if useCache {
		cache := filepath.Join(basedir, ".modelx", desc.Name+".tar.gz")
		if err := writeFile(cache, content, desc.Mode.Perm()); err != nil {
			return err
		}
		f, err := os.Open(cache)
		if err != nil {
			return err
		}
		defer f.Close()

		fi, err := os.Stat(cache)
		if err != nil {
			return err
		}
		src = bar.WrapReader(src, desc.Digest.Hex()[:8], fi.Size(), "extracting", "done", "failed")
	}
	return UnTGZ(ctx, filepath.Join(basedir, desc.Name), src)
}
