package client

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/vbauerster/mpb/v7"
	"golang.org/x/sync/errgroup"
	"kubegems.io/modelx/pkg/client/progress"
	"kubegems.io/modelx/pkg/types"
)

func (c Client) PushPack(ctx context.Context, repo, version string, pack Package) error {
	p := mpb.New(mpb.WithWidth(40))
	eg := &errgroup.Group{}
	// push all descriptors
	for _, blob := range append(pack.Blobs, pack.Config) {
		blob := blob
		eg.Go(func() error {
			if err := c.PushBlob(ctx, repo, pack.BaseDir, blob, p); err != nil {
				return err
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}
	if err := c.PutManifest(ctx, repo, version, pack.Manifest); err != nil {
		return err
	}
	progress.ShowImmediatelyProgressBar(p, types.Descriptor{Name: "manifest"}, "done")
	p.Wait()
	return nil
}

func (c Client) PushBlob(ctx context.Context, repo string, basedir string, desc types.Descriptor, p *mpb.Progress) error {
	filename := filepath.Join(basedir, desc.Name)
	fi, err := os.Stat(filename)
	if err != nil {
		return err
	}
	filesize := fi.Size()

	exist, err := c.remote.HeadBlob(ctx, repo, desc.Digest)
	if err != nil {
		return err
	}
	if exist {
		progress.ShowImmediatelyProgressBar(p, desc, "skipped")
		return nil
	}

	var bar *progress.ProgressBar
	if p != nil {
		bar = progress.NewProgressBar(p, desc, "done")
		defer bar.Close()
	}

	getbody := func() (io.ReadCloser, error) {
		f, err := os.OpenFile(filename, os.O_RDONLY, 0)
		if err != nil {
			return nil, err
		}
		readcloser := io.ReadCloser(f)
		if bar != nil {
			readcloser = bar.WrapReadCloser(filesize, readcloser, false)
		}
		return readcloser, nil
	}
	if err := c.remote.UploadBlob(ctx, repo, desc, getbody); err != nil {
		return err
	}
	bar.Done()
	return nil
}
