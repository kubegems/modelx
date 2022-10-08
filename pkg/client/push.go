package client

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/opencontainers/go-digest"
	"golang.org/x/exp/slices"
	"kubegems.io/modelx/pkg/client/progress"
	"kubegems.io/modelx/pkg/types"
)

const (
	MediaTypeModelIndexJson      = "application/vnd.modelx.model.index.v1.json"
	MediaTypeModelManifestJson   = "application/vnd.modelx.model.manifest.v1.json"
	MediaTypeModelConfigYaml     = "application/vnd.modelx.model.config.v1.yaml"
	MediaTypeModelFile           = "application/vnd.modelx.model.file.v1"
	MediaTypeModelDirectoryTarGz = "application/vnd.modelx.model.directory.v1.tar+gz"
)

var EmptyFileDigiest = digest.Canonical.FromBytes(nil)

const PushConcurrency = 5

func (c Client) Push(ctx context.Context, repo, version string, configfile, basedir string) error {
	manifest := types.Manifest{
		MediaType: MediaTypeModelManifestJson,
	}

	ds, err := os.ReadDir(basedir)
	if err != nil {
		return err
	}

	for _, entry := range ds {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if entry.Name() == configfile {
			manifest.Config = types.Descriptor{
				Name:      entry.Name(),
				MediaType: MediaTypeModelConfigYaml,
			}
			continue
		}
		if entry.IsDir() {
			manifest.Blobs = append(manifest.Blobs, types.Descriptor{
				Name:      entry.Name(),
				MediaType: MediaTypeModelDirectoryTarGz,
			})
			continue
		}
		manifest.Blobs = append(manifest.Blobs, types.Descriptor{
			Name:      entry.Name(),
			MediaType: MediaTypeModelFile,
		})
	}

	// sort blobs by name
	slices.SortFunc(manifest.Blobs, types.SortDescriptorName)

	p := progress.NewMuiltiBar(os.Stdout, 40)
	go p.Run(ctx)

	// push blobs
	for i := range manifest.Blobs {
		desc := &manifest.Blobs[i]

		p.Go(desc.Name, "pending", func(b *progress.Bar) error {
			switch desc.MediaType {
			case MediaTypeModelFile:
				return c.pushFile(ctx, basedir, desc, repo, b)
			case MediaTypeModelDirectoryTarGz:
				return c.pushDirectory(ctx, basedir, desc, repo, b)
			default:
				return nil
			}
		})

	}

	// push config
	p.Go(manifest.Config.Name, "pending", func(b *progress.Bar) error {
		return c.pushFile(ctx, basedir, &manifest.Config, repo, b)
	})

	if err := p.Wait(); err != nil {
		return err
	}

	// push manifest
	p.Go("manifest", "pushing", func(b *progress.Bar) error {
		if err := c.PutManifest(ctx, repo, version, manifest); err != nil {
			return err
		}
		b.SetStatus("manifest", "done")
		return nil
	})
	return p.Wait()
}

func (c Client) PushBlob(ctx context.Context, repo string, desc DescriptorWithContent, p *progress.Bar) error {
	if desc.Digest == EmptyFileDigiest {
		p.SetStatus(desc.Digest.Hex()[:8], "empty")
		return nil
	}

	exist, err := c.Remote.HeadBlob(ctx, repo, desc.Digest)
	if err != nil {
		return err
	}
	if exist {
		p.SetProgress(desc.Size, desc.Size)
		p.SetStatus(desc.Digest.Hex()[:8], "skipped")
		return nil
	}

	reqbody := RqeuestBody{
		ContentLength: desc.Size,
		ContentBody: func() (io.ReadCloser, error) {
			rc, err := desc.Content()
			if err != nil {
				return nil, err
			}
			return p.WrapReader(rc, desc.Digest.Hex()[:8], desc.Size, "pushing", "done", "failed"), nil
		},
	}
	return c.Remote.UploadBlob(ctx, repo, desc.Descriptor, reqbody)
}

func (c Client) pushDirectory(ctx context.Context, dir string, desc *types.Descriptor, repo string, bar *progress.Bar) error {
	tgzfile := filepath.Join(dir, ".modelx", desc.Name+".tar.gz")
	entrydir := filepath.Join(dir, desc.Name)

	fi, err := os.Stat(entrydir)
	if err != nil {
		return err
	}

	bar.SetStatus(desc.Name, "digesting")

	digest, err := TGZ(ctx, entrydir, tgzfile)
	if err != nil {
		return err
	}
	tgzfi, err := os.Stat(tgzfile)
	if err != nil {
		return err
	}

	bar.SetStatus(digest.Hex()[:8], "preparing")

	desc.Digest = digest
	desc.Size = tgzfi.Size()
	desc.Mode = fi.Mode()
	desc.Modified = fi.ModTime()

	getbody := func() (io.ReadCloser, error) {
		return os.Open(tgzfile)
	}
	return c.PushBlob(ctx, repo, DescriptorWithContent{Descriptor: *desc, Content: getbody}, bar)
}

func (c Client) pushFile(ctx context.Context, basedir string, desc *types.Descriptor, repo string, bar *progress.Bar) error {
	filename := filepath.Join(basedir, desc.Name)

	fi, err := os.Stat(filename)
	if err != nil {
		return err
	}

	f, err := os.Open(filename)
	if err != nil {
		return err
	}

	bar.SetStatus(desc.Name, "digesting")

	digest, err := digest.FromReader(f)
	_ = f.Close()
	if err != nil {
		return err
	}

	bar.SetStatus(digest.Hex()[:8], "preparing")

	desc.Digest = digest
	desc.Size = fi.Size()
	desc.Mode = fi.Mode()
	desc.Modified = fi.ModTime()

	getReader := func() (io.ReadCloser, error) {
		return os.Open(filename)
	}
	return c.PushBlob(ctx, repo, DescriptorWithContent{Descriptor: *desc, Content: getReader}, bar)
}
