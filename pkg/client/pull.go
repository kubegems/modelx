package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/opencontainers/go-digest"
	"github.com/vbauerster/mpb/v7"
	"golang.org/x/sync/errgroup"
	"kubegems.io/modelx/pkg/client/progress"
	"kubegems.io/modelx/pkg/types"
)

func Pull(ctx context.Context, ref string, into string, onprogress func(status ProgressStatistic)) error {
	reference, err := ParseReference(ref)
	if err != nil {
		return err
	}
	if reference.Repository == "" {
		return errors.New("repository is not specified")
	}
	if into == "" {
		into = path.Base(reference.Repository)
	}
	fmt.Printf("Pulling %s into %s \n", ref, into)

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

	p := mpb.New(mpb.WithWidth(40))

	remote := RegistryClient{
		Client: &http.Client{},
		Addr:   reference.Registry,
	}

	manifest, err := remote.GetManifest(ctx, reference.Repository, reference.Version)
	if err != nil {
		return err
	}

	progress.ShowImmediatelyProgressBar(p, types.Descriptor{Name: "manifest"}, "done")

	eg := &errgroup.Group{}
	for _, blob := range manifest.Blobs {
		blob := blob
		eg.Go(func() error {
			return PullBlob(ctx, reference, into, blob, p)
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}
	p.Wait()
	return nil
}

func PullBlob(ctx context.Context, reference Reference, into string, desc types.Descriptor, p *mpb.Progress) error {
	remote := RegistryClient{
		Client: &http.Client{},
		Addr:   reference.Registry,
	}
	// check local file
	localfilename := filepath.Join(into, desc.Name)

	// file exists, check hash
	if f, err := os.OpenFile(localfilename, os.O_RDONLY, 0); err == nil {
		defer f.Close()

		digest, err := digest.FromReader(f)
		if err != nil {
			return err
		}
		if digest.String() == desc.Digest.String() {
			progress.ShowImmediatelyProgressBar(p, desc, "already exists")
			return nil
		}
		// file exists but hash is not correct
	}

	content, len, err := remote.GetBlob(ctx, reference.Repository, desc.Digest)
	if err != nil {
		return err
	}
	defer content.Close()

	bar := progress.CreateProgressBar(p, desc, "done")
	defer bar.Close()

	reader := bar.Reader(len, content)
	if err := WriteBlob(ctx, localfilename, reader); err != nil {
		return err
	}
	return nil
}

func WriteBlob(ctx context.Context, filename string, content io.Reader) error {
	// check parent directory
	dir := filepath.Dir(filename)
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	// write file
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, content)
	if err != nil {
		return err
	}
	return nil
}
