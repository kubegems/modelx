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

const PullPushConcurrency = 3

func (c Client) Push(ctx context.Context, repo, version string, configfile, basedir string) error {
	manifest, err := ParseManifest(ctx, basedir, configfile)
	if err != nil {
		return err
	}
	p := progress.NewMuiltiBar(os.Stdout, 40, PullPushConcurrency)
	go p.Run(ctx)

	// push blobs
	for i := range manifest.Blobs {
		desc := &manifest.Blobs[i]
		p.Go(desc.Name, "pending", func(b *progress.Bar) error {
			switch desc.MediaType {
			case MediaTypeModelFile:
				return c.pushFile(ctx, filepath.Join(basedir, desc.Name), desc, repo, b)
			case MediaTypeModelDirectoryTarGz:
				return c.pushDirectory(ctx, basedir, filepath.Join(basedir, desc.Name), desc, repo, b)
			default:
				return nil
			}
		})
	}
	// push config
	p.Go(manifest.Config.Name, "pending", func(b *progress.Bar) error {
		return c.pushFile(ctx, filepath.Join(basedir, manifest.Config.Name), &manifest.Config, repo, b)
	})
	if err := p.Wait(); err != nil {
		return err
	}
	// push manifest
	p.Go("manifest", "pushing", func(b *progress.Bar) error {
		if err := c.PutManifest(ctx, repo, version, *manifest); err != nil {
			return err
		}
		b.SetNameStatus("manifest", "done")
		return nil
	})
	return p.Wait()
}

func ParseManifest(ctx context.Context, basedir string, configfile string) (*types.Manifest, error) {
	manifest := &types.Manifest{
		MediaType: MediaTypeModelManifestJson,
	}
	ds, err := os.ReadDir(basedir)
	if err != nil {
		return nil, err
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
	slices.SortFunc(manifest.Blobs, types.SortDescriptorName)
	return manifest, nil
}

func (c Client) pushDirectory(ctx context.Context, cachedir, blobdir string, desc *types.Descriptor, repo string, bar *progress.Bar) error {
	diri, err := os.Stat(blobdir)
	if err != nil {
		return err
	}
	desc.Mode = diri.Mode()
	desc.Modified = diri.ModTime()

	bar.SetNameStatus(desc.Name, "digesting")
	filename := filepath.Join(cachedir, ".modelx", desc.Name+".tar.gz")
	digest, err := TGZ(ctx, blobdir, filename)
	if err != nil {
		return err
	}
	desc.Digest = digest
	return c.pushFile(ctx, filename, desc, repo, bar)
}

func (c Client) pushFile(ctx context.Context, blobfile string, desc *types.Descriptor, repo string, bar *progress.Bar) error {
	fi, err := os.Stat(blobfile)
	if err != nil {
		return err
	}
	if desc.Digest == "" {
		bar.SetNameStatus(desc.Name, "digesting")
		f, err := os.Open(blobfile)
		if err != nil {
			return err
		}
		digest, err := digest.FromReader(f)
		_ = f.Close()
		if err != nil {
			return err
		}
		desc.Digest = digest
	}
	if desc.Size == 0 {
		desc.Size = fi.Size()
	}
	if desc.Mode == 0 {
		desc.Mode = fi.Mode()
	}
	if desc.Modified.IsZero() {
		desc.Modified = fi.ModTime()
	}
	getReader := func() (io.ReadCloser, error) {
		return os.Open(blobfile)
	}
	bar.SetNameStatus(desc.Digest.Hex()[:8], "pending")
	return c.PushBlob(ctx, repo, DescriptorWithContent{Descriptor: *desc, GetContent: getReader}, bar)
}

func (c Client) PushBlob(ctx context.Context, repo string, desc DescriptorWithContent, p *progress.Bar) error {
	if desc.Digest == EmptyFileDigiest {
		p.SetStatus("empty")
		return nil
	}
	exist, err := c.Remote.HeadBlob(ctx, repo, desc.Digest)
	if err != nil {
		return err
	}
	if exist {
		p.SetProgress(desc.Size, desc.Size)
		p.SetStatus("skipped")
		return nil
	}
	wrappdesc := DescriptorWithContent{
		Descriptor: desc.Descriptor,
		GetContent: func() (io.ReadCloser, error) {
			content, err := desc.GetContent()
			if err != nil {
				return nil, err
			}
			content = p.WrapReader(content, desc.Size, "pushing", "failed")
			return content, nil
		},
	}
	if err := c.Remote.UploadBlobContent(ctx, repo, wrappdesc); err != nil {
		return err
	}
	p.SetStatus("done")
	return nil
}
