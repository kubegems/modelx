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

const PushConcurrency = 5

func (c Client) Push(ctx context.Context, repo, version string, manifest types.Manifest, basedir string) error {
	p := mpb.New(mpb.WithWidth(40))

	eg := errgroup.Group{}
	eg.SetLimit(PushConcurrency)
	for i := range manifest.Blobs {
		i := i
		eg.Go(func() error {
			return c.PushBlob(ctx, repo, basedir, &manifest.Blobs[i], p)
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}
	if err := c.PushBlob(ctx, repo, basedir, &manifest.Config, p); err != nil {
		return err
	}
	if err := c.PutManifest(ctx, repo, version, manifest); err != nil {
		return err
	}
	progress.ShowImmediatelyProgressBar(p, types.Descriptor{Name: "manifest"}, "done")
	p.Wait()
	return nil
}

func (c Client) PushBlob(ctx context.Context, repo string, basedir string, desc *types.Descriptor, p *mpb.Progress) error {
	if desc.Name == "" {
		return fmt.Errorf("empty filename")
	}
	filename := filepath.Join(basedir, desc.Name)
	fi, err := os.Stat(filename)
	if err != nil {
		return err
	}
	desc.Modified = fi.ModTime()
	desc.Size = fi.Size()

	// calc digest
	if desc.Digest == "" {
		f, err := os.Open(filename)
		if err != nil {
			return err
		}
		desc.Digest, err = digest.FromReader(f)
		if err != nil {
			_ = f.Close()
			return err
		}
		_ = f.Close()
	}

	exist, err := c.remote.HeadBlob(ctx, repo, desc.Digest)
	if err != nil {
		return err
	}
	if exist {
		progress.ShowImmediatelyProgressBar(p, *desc, "skipped")
		return nil
	}

	var bar *progress.ProgressBar
	if p != nil {
		bar = progress.NewProgressBar(p, *desc, "done")
		defer bar.Close()
	}

	getbody := func() (io.ReadCloser, error) {
		f, err := os.OpenFile(filename, os.O_RDONLY, 0)
		if err != nil {
			return nil, err
		}
		readcloser := io.ReadCloser(f)
		if bar != nil {
			readcloser = bar.WrapReadCloser(desc.Size, readcloser, false)
		}
		return readcloser, nil
	}

	reqbody := RqeuestBody{
		ContentLength: desc.Size,
		ContentBody:   getbody,
	}
	if err := c.remote.UploadBlob(ctx, repo, *desc, reqbody); err != nil {
		return err
	}
	bar.Done()
	return nil
}
