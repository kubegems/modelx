package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/opencontainers/go-digest"
	"github.com/vbauerster/mpb/v7"
	"golang.org/x/sync/errgroup"
	"kubegems.io/modelx/pkg/client/progress"
	"kubegems.io/modelx/pkg/types"
)

func PullPack(ctx context.Context, reference Reference, pack Package) error {
	into := pack.BaseDir

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

	fmt.Printf("Pulling %s into %s \n", reference.String(), into)
	p := mpb.New(mpb.WithWidth(40))

	eg := &errgroup.Group{}
	for _, blob := range append(pack.Blobs, pack.Config) {
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
			f, err := prepareWritefile(filepath.Join(into, blob.Name))
			if err != nil {
				return err
			}
			defer f.Close()

			return PullBlob(ctx, reference, f, blob, p)
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
	// file exists, check hash
	if f, err := os.OpenFile(localfilename, os.O_RDONLY, 0); err == nil {
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

func PullBlob(ctx context.Context, reference Reference, into io.Writer, desc types.Descriptor, p *mpb.Progress) error {
	remote := RegistryClient{
		Client: &http.Client{},
		Addr:   reference.Registry,
	}

	content, len, err := remote.GetBlob(ctx, reference.Repository, desc.Digest)
	if err != nil {
		return err
	}
	defer content.Close()

	rc := io.NopCloser(io.Reader(content))
	defer rc.Close()

	if p != nil {
		bar := progress.NewProgressBar(p, desc, "done")
		defer bar.Close()
		rc = bar.WrapReadCloser(len, rc, true)
	}

	if _, err := io.Copy(into, rc); err != nil {
		return err
	}

	return nil
}
