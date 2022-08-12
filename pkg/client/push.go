package client

import (
	"context"
	"fmt"

	"github.com/vbauerster/mpb/v7"
	"golang.org/x/exp/slices"
	"golang.org/x/sync/errgroup"
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

const PushConcurrency = 5

func (c Client) Push(ctx context.Context, repo, version string, configfile, basedir string) error {
	p := mpb.New(mpb.WithWidth(40))

	manifest := types.Manifest{
		MediaType: MediaTypeModelManifestJson,
	}
	bodymap, err := ParseDir(ctx, basedir)
	if err != nil {
		return err
	}
	for name, item := range bodymap {
		if name == configfile {
			manifest.Config = item.Descriptor
			manifest.Config.MediaType = MediaTypeModelConfigYaml
		} else {
			manifest.Blobs = append(manifest.Blobs, item.Descriptor)
		}
	}
	slices.SortFunc(manifest.Blobs, types.SortDescriptorName)

	eg := errgroup.Group{}
	eg.SetLimit(PushConcurrency)
	for _, blob := range append(manifest.Blobs, manifest.Config) {
		i := blob
		eg.Go(func() error {
			content, ok := bodymap[i.Name]
			if !ok {
				return fmt.Errorf("missing content for %s", i.Name)
			}
			return c.PushBlob(ctx, repo, content, p)
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}
	if err := c.PutManifest(ctx, repo, version, manifest); err != nil {
		return err
	}
	progress.ShowImmediatelyProgressBar(p, types.Descriptor{Name: "manifest"}, "done")
	p.Wait()
	return nil
}

func (c Client) PushBlob(ctx context.Context, repo string, desc DescriptorWithContent, p *mpb.Progress) error {
	exist, err := c.remote.HeadBlob(ctx, repo, desc.Digest)
	if err != nil {
		return err
	}
	if exist {
		progress.ShowImmediatelyProgressBar(p, desc.Descriptor, "skipped")
		return nil
	}
	var bar *progress.ProgressBar
	if p != nil {
		bar = progress.NewProgressBar(p, desc.Descriptor, "done")
		defer bar.Close()
	}
	reqbody := RqeuestBody{
		ContentLength: desc.Size,
		ContentBody:   desc.Content,
	}
	if err := c.remote.UploadBlob(ctx, repo, desc.Descriptor, reqbody); err != nil {
		return err
	}
	bar.Done()
	return nil
}
