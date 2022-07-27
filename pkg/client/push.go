package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/vbauerster/mpb/v7"
	"golang.org/x/sync/errgroup"
	"kubegems.io/modelx/pkg/client/progress"
	"kubegems.io/modelx/pkg/types"
)

type ProgressStatus string

const (
	ProgressStatusPending ProgressStatus = "pending"
	ProgressStatusPulling ProgressStatus = "pulling"
	ProgressStatusSkipped ProgressStatus = "skipped"
	ProgressStatusPushing ProgressStatus = "pushing"
	ProgressStatusFailed  ProgressStatus = "failed"
	ProgressStatusDone    ProgressStatus = "done"
)

type ProgressStatistic struct {
	Name   string
	Status ProgressStatus
	Count  int64
	Total  int64
	Done   bool
	Failed bool
}

func Push(ctx context.Context, ref string, dir string) error {
	reference, err := ParseReference(ref)
	if err != nil {
		return err
	}
	if reference.Repository == "" {
		return errors.New("repository is not specified")
	}

	localmodel, err := PackLocalModel(ctx, dir)
	if err != nil {
		return err
	}

	fmt.Printf("Pushing to %s \n", ref)

	p := mpb.New(mpb.WithWidth(40))

	eg := &errgroup.Group{}
	for _, blob := range localmodel.Manifest.Blobs {
		blob := blob
		eg.Go(func() error {
			if err := PushBlob(ctx, reference, localmodel.BaseDir, blob, p); err != nil {
				return err
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}
	if err := PushManifest(ctx, reference, localmodel.Manifest, p); err != nil {
		return err
	}
	p.Wait()
	return nil
}

func PushManifest(ctx context.Context, ref Reference, manifest types.Manifest, p *mpb.Progress) error {
	remote := RegistryClient{
		Client: &http.Client{},
		Addr:   ref.Registry,
	}
	bar := progress.CreateProgressBar(p, types.Descriptor{Name: "manifest"}, "done")
	defer bar.Close()
	if err := remote.PutManifest(ctx, ref.Repository, ref.Version, manifest); err != nil {
		return err
	}
	bar.Complete()
	return nil
}

func PushBlob(ctx context.Context, ref Reference, basedir string, desc types.Descriptor, p *mpb.Progress) error {
	filename := filepath.Join(basedir, desc.Name)
	f, err := os.OpenFile(filename, os.O_RDONLY, 0)
	if err != nil {
		return err
	}

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

	bar := progress.CreateProgressBar(p, desc, "done")
	defer bar.Close()

	reader := bar.Reader(filesize, f)

	if err := remote.UploadBlob(ctx, ref.Repository, desc, reader); err != nil {
		return err
	}
	return nil
}

type CountWriter struct {
	written  int64
	OnChange func(written int64)
}

func (cw *CountWriter) Write(p []byte) (int, error) {
	n := len(p)
	cw.written += int64(n)
	if cw.OnChange != nil {
		go cw.OnChange(cw.written)
	}
	return n, nil
}
