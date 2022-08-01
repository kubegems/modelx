package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/vbauerster/mpb/v7"
	"golang.org/x/sync/errgroup"
	"kubegems.io/modelx/pkg/client/progress"
	"kubegems.io/modelx/pkg/types"
)

func PushPack(ctx context.Context, ref Reference, pack Package) error {
	fmt.Printf("Pushing to %s \n", ref.String())
	if ref.Repository == "" {
		return errors.New("repository is not specified")
	}

	p := mpb.New(mpb.WithWidth(40))
	eg := &errgroup.Group{}
	// push all descriptors
	for _, blob := range append(pack.Blobs, pack.Config) {
		blob := blob
		eg.Go(func() error {
			if err := PushBlob(ctx, ref, pack.BaseDir, blob, p); err != nil {
				return err
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}
	if err := PushManifest(ctx, ref, pack.Manifest, p); err != nil {
		return err
	}
	progress.ShowImmediatelyProgressBar(p, types.Descriptor{Name: "manifest"}, "done")
	p.Wait()
	return nil
}

func PushManifest(ctx context.Context, ref Reference, manifest types.Manifest, p *mpb.Progress) error {
	remote := RegistryClient{
		Client: &http.Client{},
		Addr:   ref.Registry,
	}
	if err := remote.PutManifest(ctx, ref.Repository, ref.Version, manifest); err != nil {
		return err
	}
	return nil
}

func PushBlob(ctx context.Context, ref Reference, basedir string, desc types.Descriptor, p *mpb.Progress) error {
	filename := filepath.Join(basedir, desc.Name)
	fi, err := os.Stat(filename)
	if err != nil {
		return err
	}
	filesize := fi.Size()

	remote := RegistryClient{
		Client: &http.Client{},
		Addr:   ref.Registry,
	}

	exist, err := remote.HeadBlob(ctx, ref.Repository, desc.Digest)
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
	if err := remote.UploadBlob(ctx, ref.Repository, desc, getbody); err != nil {
		return err
	}
	bar.Done()
	return nil
}
