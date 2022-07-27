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

func CreateProgressBar(pool *mpb.Progress, desc types.Descriptor, complete string) ProgressBar {
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
	return ProgressBar{bar: bar}
}

type ProgressBar struct {
	bar *mpb.Bar
}

func (p ProgressBar) Reader(total int64, r io.Reader) io.Reader {
	p.bar.SetTotal(total, false)
	p.bar.EnableTriggerComplete()
	return p.bar.ProxyReader(r)
}

func (p ProgressBar) Complete() {
	p.bar.SetTotal(-1, true)
}

func (p ProgressBar) Close() {
	p.bar.Abort(false)
}
