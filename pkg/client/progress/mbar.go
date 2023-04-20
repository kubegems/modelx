package progress

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

type MultiBar struct {
	w               io.Writer // writer to destination
	width           int
	lastWrittenRows int
	bars            []*Bar
	barslock        sync.Mutex
	eg              *errgroup.Group

	haschange bool
}

func NewMuiltiBar(dest io.Writer, width int, concurent int) *MultiBar {
	mb := &MultiBar{
		width: width,
		w:     dest,
		eg:    &errgroup.Group{},
	}
	if concurent == 0 {
		mb.eg.SetLimit(5)
	}
	return mb
}

func (m *MultiBar) print() {
	m.barslock.Lock()
	defer m.barslock.Unlock()

	buf := &bytes.Buffer{}

	// clear previous rows
	if m.lastWrittenRows > 0 {
		fmt.Fprintf(buf, "\033[%dA\033[J", m.lastWrittenRows)
	}

	for _, b := range m.bars {
		b.Write(buf)
	}

	// write once
	_, _ = m.w.Write(buf.Bytes())
	m.lastWrittenRows = len(m.bars)
}

func (m *MultiBar) Run(ctx context.Context) {
	t := time.NewTicker(100 * time.Millisecond)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if m.haschange {
				m.haschange = false
				m.print()
			}
		}
	}
}

func (m *MultiBar) Go(name string, initstatus string, fun func(b *Bar) error) {
	bar := &Bar{
		mp:     m,
		Name:   name,
		Status: initstatus,
		Width:  m.width,
	}
	m.barslock.Lock()
	m.bars = append(m.bars, bar)
	m.barslock.Unlock()
	m.print()

	m.eg.Go(func() error {
		if err := fun(bar); err != nil {
			bar.Status = "failed"
			bar.Notify()
			return err
		}
		bar.Done = true
		bar.Notify()
		return nil
	})
}

func (m *MultiBar) Wait() error {
	// wait all goroutines to finish
	return m.eg.Wait()
}
