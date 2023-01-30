package progress

import (
	"fmt"
	"io"
	"math"
	"strings"

	"kubegems.io/modelx/pkg/client/units"
)

var SpinnerDefault = []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}

type Bar struct {
	Name      string
	Total     int64  // total bytes, -1 for indeterminate
	Completed int64  // completed bytes
	Width     int    // width of the bar
	Status    string // status text
	Done      bool   // if the bar is done
	mp        *MultiBar
}

func (b *Bar) Write(w io.Writer) {
	if b.Width == 0 {
		b.Width = 40
	}
	var completed int
	var status string

	if b.Done {
		completed = b.Width
		status = b.Status
	} else {
		if b.Total <= 0 {
			completed = 0
			status = b.Status
		} else {
			completed = int(float64(b.Width) * float64(b.Completed) / float64(b.Total))
			if completed < 0 {
				completed = 0
			}
			status = units.HumanSize(float64(b.Completed)) + "/" + units.HumanSize(float64(b.Total))
		}
	}

	fmt.Fprintf(w, "%s [%s%s] %s\n",
		b.Name,
		strings.Repeat("+", completed),
		strings.Repeat("-", b.Width-completed),
		status,
	)
}

func percent(total, completed int64) int {
	if total <= 0 {
		return 0
	}
	if completed >= total {
		return 100
	}
	round := math.Round(float64(completed) / float64(total) * 100)
	if round > 100 {
		return 100
	}
	if round < 0 {
		return 0
	}
	return int(round)
}

func (b *Bar) SetProgress(completed, total int64) {
	b.Completed, b.Total = completed, total
	b.Notify()
}

func (b *Bar) SetStatus(name, status string) {
	b.Name, b.Status = name, status
	b.Notify()
}

func (b *Bar) Increment(n int64) {
	b.Completed += n
	b.Notify()
}

func (b *Bar) WrapReader(rc io.ReadSeekCloser, name string, total int64, onProcess, onComplete, onFailed string) io.ReadSeekCloser {
	b.Total = total
	b.Status = onProcess
	b.Name = name
	defer b.Notify()
	return &barReader{rc: rc, b: b, onComplete: onComplete}
}

type barReader struct {
	rc         io.ReadSeekCloser
	b          *Bar
	onComplete string
	onFailed   string
}

func (r *Bar) Notify() {
	if r.mp != nil {
		r.mp.print()
	}
}

func (r *barReader) Read(p []byte) (int, error) {
	n, err := r.rc.Read(p)

	r.b.Completed += int64(n)
	r.b.mp.haschange = true
	if r.b.Completed >= r.b.Total {
		r.b.Status = r.onComplete
		r.b.Done = true
	}

	return n, err
}

func (r *barReader) Seek(offset int64, whence int) (int64, error) {
	return r.rc.Seek(offset, whence)
}

func (r *barReader) Close() error {
	if r.b.Completed < r.b.Total {
		r.b.Status = r.onFailed
	}
	r.b.Notify()
	return r.rc.Close()
}

func (b *Bar) WrapWriter(w io.Writer, name string, total int64, onProcess, onComplete, onFailed string) io.Writer {
	b.Name = name
	b.Total = total
	b.Status = onProcess
	b.Notify()
	return &bario{w: w, b: b, onComplete: onComplete, onFailed: onFailed}
}

type bario struct {
	w          io.Writer
	b          *Bar
	onComplete string
	onFailed   string
}

func (r *bario) Write(p []byte) (int, error) {
	n, err := r.w.Write(p)
	if err != nil {
		r.b.mp.haschange = true
		r.b.Done = true
		r.b.Status = r.onFailed
		return n, err
	}
	r.b.Completed += int64(n)
	r.b.mp.haschange = true
	if r.b.Completed >= r.b.Total {
		r.b.Status = r.onComplete
		r.b.Done = true
	}
	return n, nil
}

func (r *bario) WriteAt(p []byte, off int64) (int, error) {
	wat, ok := r.w.(io.WriterAt)
	if !ok {
		return 0, io.ErrUnexpectedEOF
	}
	n, err := wat.WriteAt(p, off)
	if err != nil {
		r.b.mp.haschange = true
		r.b.Done = true
		r.b.Status = r.onFailed
		return n, err
	}
	r.b.Completed += int64(n)
	r.b.mp.haschange = true
	if r.b.Completed >= r.b.Total {
		r.b.Status = r.onComplete
		r.b.Done = true
	}
	return n, nil
}
