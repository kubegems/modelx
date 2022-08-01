package progress

import (
	"io"

	"github.com/vbauerster/mpb/v7"
	"github.com/vbauerster/mpb/v7/decor"
	"kubegems.io/modelx/pkg/types"
)

func ShowImmediatelyProgressBar(pool *mpb.Progress, desc types.Descriptor, complete string) {
	name := desc.Name
	if len(desc.Digest) != 0 {
		name = desc.Digest.Hex()[:8]
	}
	bar := pool.AddBar(0,
		mpb.PrependDecorators(
			decor.Name(name),
		),
		mpb.AppendDecorators(
			decor.OnComplete(decor.Name(""), complete),
		),
	)
	bar.SetTotal(-1, true)
	bar.Abort(false)
}

func NewProgressBar(pool *mpb.Progress, desc types.Descriptor, complete string) *ProgressBar {
	name := desc.Name
	if len(desc.Digest) != 0 {
		name = desc.Digest.Hex()[:8]
	}
	bar := pool.AddBar(0,
		mpb.PrependDecorators(
			decor.Name(name),
		),
		mpb.AppendDecorators(
			decor.OnComplete(decor.CountersKibiByte(""), complete),
		),
	)
	return &ProgressBar{bar: bar}
}

type ProgressBar struct {
	bar *mpb.Bar
}

func (p *ProgressBar) WrapReadCloser(total int64, rc io.ReadCloser, triggerdone bool) io.ReadCloser {
	p.bar.SetTotal(total, false)

	if triggerdone {
		p.bar.EnableTriggerComplete()
	}

	return &ReadCloser{
		rc:  rc,
		bar: p.bar,
	}
}

func (p *ProgressBar) Done() {
	p.bar.SetTotal(p.bar.Current(), true)
}

func (p *ProgressBar) Close() error {
	p.bar.Abort(false)
	return nil
}

type ReadCloser struct {
	bar *mpb.Bar
	rc  io.ReadCloser
}

func (r *ReadCloser) Read(p []byte) (int, error) {
	n, err := r.rc.Read(p)
	r.bar.IncrBy(n)
	return n, err
}

func (r *ReadCloser) Close() error {
	return r.rc.Close()
}
