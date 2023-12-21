package progress

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"golang.org/x/term"
)

type MultiBar struct {
	cancel          context.CancelFunc
	w               io.Writer // writer to destination
	width           int
	nameWidth       int
	lastWrittenRows int
	bars            []*Bar
	barslock        sync.Mutex
	eg              *errgroup.Group
	haschange       bool // state machine to reduce print
}

func NewMuiltiBarContext(ctx context.Context, dest io.Writer, width int, concurent int) (*MultiBar, context.Context) {
	if width <= 0 {
		w, _, err := term.GetSize(0)
		if err == nil {
			if w < 40 {
				w = 40
			}
			width = w - 20 // min 20 chars for status
		} else {
			width = 60
		}
	}
	ctx, cancel := context.WithCancel(ctx)
	mb := &MultiBar{
		width:     width,
		nameWidth: 8,
		w:         dest,
		eg:        &errgroup.Group{}, // eg's context canceld on Wait() called, but we won't.
		cancel:    cancel,
	}
	if concurent == 0 {
		mb.eg.SetLimit(5)
	} else {
		mb.eg.SetLimit(concurent)
	}
	go mb.run(ctx)
	return mb, ctx
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
		b.Print(buf)
	}

	// write once
	_, _ = m.w.Write(buf.Bytes())
	m.lastWrittenRows = len(m.bars)
}

func (m *MultiBar) run(ctx context.Context) {
	t := time.NewTicker(100 * time.Millisecond)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			// print once
			m.print()
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
		mp:         m,
		Name:       name,
		Status:     initstatus,
		Width:      m.width,
		MaxNameLen: m.nameWidth,
	}
	m.barslock.Lock()
	m.bars = append(m.bars, bar)
	m.barslock.Unlock()
	m.print()

	m.eg.Go(func() error {
		if err := fun(bar); err != nil {
			bar.Status = "failed"
			bar.Notify()
			// cancel all other bars
			m.cancel()
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
