package client

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/opencontainers/go-digest"
	"github.com/vbauerster/mpb/v7"
	"golang.org/x/sync/errgroup"
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

	p := mpb.New(mpb.WithWidth(40))

	eg := &errgroup.Group{}
	for _, blob := range append(manifest.Blobs, manifest.Config) {
		blob := blob
		eg.Go(func() error {
			ok, err := checkLocalBlob(ctx, into, blob)
			if err != nil {
				return err
			}
			if ok {
				progress.ShowImmediatelyProgressBar(p, blob, "already exists")
				return nil
			}
			readercloser, err := c.PullBlob(ctx, repo, blob, p)
			if err != nil {
				return err
			}
			defer readercloser.Close()

			switch blob.MediaType {
			case MediaTypeModelDirectoryTarGz:
				basedir := filepath.Join(into, blob.Name)
				return UnTGZ(ctx, basedir, readercloser)
			default:
				f, err := os.Create(filepath.Join(into, blob.Name))
				if err != nil {
					return err
				}
				defer f.Close()
				_, err = io.Copy(f, readercloser)
				return err
			}
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}
	p.Wait()
	return nil
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

func prepareWritefile(filename string) (*os.File, error) {
	// check parent directory
	dir := filepath.Dir(filename)
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (c Client) PullBlob(ctx context.Context, repo string, desc types.Descriptor, p *mpb.Progress) (io.ReadCloser, error) {
	content, len, err := c.remote.GetBlob(ctx, repo, desc.Digest)
	if err != nil {
		return nil, err
	}

	if p != nil {
		bar := progress.NewProgressBar(p, desc, "done")
		defer bar.Close()
		content = bar.WrapReadCloser(len, content, true)
	}

	return content, nil
}
