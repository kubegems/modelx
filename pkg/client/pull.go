package client

import (
	"context"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/opencontainers/go-digest"
	"golang.org/x/sync/errgroup"
	"kubegems.io/modelx/pkg/client/progress"
	"kubegems.io/modelx/pkg/errors"
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
	return c.PullBlobs(ctx, repo, into, append(manifest.Blobs, manifest.Config))
}

func (c Client) PullBlobs(ctx context.Context, repo string, basedir string, blobs []types.Descriptor) error {
	mb, ctx := progress.NewMuiltiBarContext(ctx, os.Stdout, 60, PullPushConcurrency)
	for _, blob := range blobs {
		blob := blob
		mb.Go(blob.Name, "pending", func(b *progress.Bar) error {
			return c.pullBlobProgress(ctx, repo, blob, basedir, b)
		})
	}
	return mb.Wait()
}

func (c Client) pullBlobProgress(ctx context.Context, repo string, desc types.Descriptor, basedir string, bar *progress.Bar) error {
	switch desc.MediaType {
	case MediaTypeModelDirectoryTarGz:
		return c.pullDirectory(ctx, repo, desc, basedir, bar, true)
	case MediaTypeModelFile:
		return c.pullFile(ctx, repo, desc, basedir, bar)
	case MediaTypeModelConfigYaml:
		return c.pullConfig(ctx, repo, desc, basedir, bar)
	default:
		return fmt.Errorf("unsupported media type %s", desc.MediaType)
	}
}

func OpenWriteFile(filename string, perm os.FileMode) (*os.File, error) {
	if perm == 0 {
		perm = 0o644
	}
	if err := os.MkdirAll(filepath.Dir(filename), os.ModePerm); err != nil {
		return nil, err
	}
	return os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm.Perm())
}

func WriteToFile(filename string, src io.Reader, perm os.FileMode) error {
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

	if closer, ok := src.(io.Closer); ok {
		defer closer.Close()
	}

	_, err = io.Copy(f, src)
	return err
}

func (c Client) pullConfig(ctx context.Context, repo string, desc types.Descriptor, basedir string, bar *progress.Bar) error {
	return c.pullFile(ctx, repo, desc, basedir, bar)
}

func (c Client) pullFile(ctx context.Context, repo string, desc types.Descriptor, basedir string, bar *progress.Bar) error {
	// check hash
	bar.SetNameStatus(desc.Name, "checking", false)
	filename := filepath.Join(basedir, desc.Name)
	if f, err := os.Open(filename); err == nil {
		digest, err := digest.FromReader(f)
		if err != nil {
			return err
		}
		if digest.String() == desc.Digest.String() {
			bar.SetNameStatus(desc.Digest.Hex()[:8], "already exists", true)
			return nil
		}
		_ = f.Close()
	} else if !os.IsNotExist(err) {
		return err
	}

	f, err := OpenWriteFile(filename, desc.Mode.Perm())
	if err != nil {
		return err
	}
	defer f.Close()
	if desc.Digest == EmptyFileDigiest {
		return nil
	}
	w := bar.WrapWriter(f, desc.Digest.Hex()[:8], desc.Size, "downloading")
	if err := c.PullBlob(ctx, repo, desc, w); err != nil {
		return err
	}
	bar.SetStatus("done", true)
	return nil
}

func (c Client) pullDirectory(ctx context.Context, repo string, desc types.Descriptor, basedir string, bar *progress.Bar, useCache bool) error {
	// check hash
	bar.SetNameStatus(desc.Name, "checking", false)
	digest, err := TGZ(ctx, filepath.Join(basedir, desc.Name), "")
	if err != nil {
		return err
	}
	if digest.String() == desc.Digest.String() {
		bar.SetNameStatus(desc.Digest.Hex()[:8], "already exists", true)
		return nil
	}

	// pull to cache
	if useCache {
		cache := filepath.Join(basedir, ".modelx", desc.Name+".tar.gz")
		wf, err := OpenWriteFile(cache, desc.Mode)
		if err != nil {
			return err
		}
		defer wf.Close()

		w := bar.WrapWriter(wf, desc.Digest.Hex()[:8], desc.Size, "downloading")
		if err := c.PullBlob(ctx, repo, desc, w); err != nil {
			return err
		}
		_ = wf.Close()

		// extract
		rf, err := os.Open(cache)
		if err != nil {
			return err
		}
		r := bar.WrapReader(rf, desc.Digest.Hex()[:8], desc.Size, "extracting")
		if err := UnTGZ(ctx, filepath.Join(basedir, desc.Name), r); err != nil {
			return err
		}
		bar.SetStatus("done", true)
		return nil
	} else {
		// download and extract at same time
		piper, pipew := io.Pipe()
		var src io.Reader = piper

		eg, ctx := errgroup.WithContext(ctx)
		// download
		eg.Go(func() error {
			w := bar.WrapWriter(pipew, desc.Digest.Hex()[:8], desc.Size, "downloading")
			return c.PullBlob(ctx, repo, desc, w)
		})
		// extract
		eg.Go(func() error {
			if err := UnTGZ(ctx, filepath.Join(basedir, desc.Name), src); err != nil {
				return err
			}
			bar.SetStatus("done", true)
			return nil
		})
		return eg.Wait()
	}
}

func (c Client) PullBlob(ctx context.Context, repo string, desc types.Descriptor, into io.Writer) error {
	location, err := c.Remote.GetBlobLocation(ctx, repo, desc, types.BlobLocationPurposeDownload)
	if err != nil {
		if !IsServerUnsupportError(err) {
			return err
		}
		return c.Remote.GetBlobContent(ctx, repo, desc.Digest, into)
	}
	return c.Extension.Download(ctx, desc, *location, into)
}

func IsServerUnsupportError(err error) bool {
	info := errors.ErrorInfo{}
	if stderrors.As(err, &info) {
		return info.Code == errors.ErrCodeUnsupported || info.HttpStatus == http.StatusNotFound
	}
	return false
}
