package progress

import (
	"io"

	"github.com/google/uuid"
)

func (b *Bar) WrapReader(rc io.ReadSeekCloser, name string, total int64, initStatus string) io.ReadSeekCloser {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.Total = total
	b.Name = name
	b.Status = initStatus
	b.Done = false // reset done
	defer b.Notify()

	if b.Fragments == nil {
		b.Fragments = make(map[string]*BarFragment)
	}
	uid := uuid.New().String()
	thisfragment := &BarFragment{
		uid: uid,
	}
	b.Fragments[uid] = thisfragment

	return &barr{fragment: thisfragment, rc: rc, b: b}
}

type barr struct {
	fragment *BarFragment
	rc       io.ReadSeekCloser
	haserr   bool
	b        *Bar
}

func (r *barr) Seek(offset int64, whence int) (int64, error) {
	r.fragment.Processed = 0 // reset processed
	n, err := r.rc.Seek(offset, whence)
	if err != nil {
		r.b.Status = "failed"
		r.b.Done = true
	}
	switch whence {
	case io.SeekStart:
		r.fragment.Offset = n
	case io.SeekCurrent:
		r.fragment.Offset += n
	case io.SeekEnd:
		r.fragment.Offset = r.b.Total - n
	}
	r.b.mp.haschange = true
	return n, err
}

func (r *barr) Read(p []byte) (int, error) {
	n, err := r.rc.Read(p)
	if err != nil && err != io.EOF {
		r.b.Status = "failed"
		r.b.Done = true
		r.haserr = true
	}
	r.fragment.Processed += int64(n)
	r.b.mp.haschange = true
	return n, err
}

func (r *barr) Close() error {
	if r.haserr {
		r.b.mu.Lock()
		defer r.b.mu.Unlock()
		delete(r.b.Fragments, r.fragment.uid)
	}

	return r.rc.Close()
}

func (b *Bar) WrapWriter(wc io.WriteCloser, name string, total int64, initStatus string) io.WriteCloser {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.Name = name
	b.Total = total
	b.Status = initStatus
	b.Notify()

	if b.Fragments == nil {
		b.Fragments = make(map[string]*BarFragment)
	}
	uid := uuid.New().String()
	thisfragment := &BarFragment{
		uid: uid,
	}
	b.Fragments[uid] = thisfragment

	w := &barw{fragment: thisfragment, wc: wc, b: b}
	if _, ok := wc.(io.WriterAt); ok {
		return barwa{barw: w}
	}
	return w
}

type barw struct {
	fragment *BarFragment
	wc       io.WriteCloser
	b        *Bar
	haserr   bool
}

func (r *barw) Write(p []byte) (int, error) {
	n, err := r.wc.Write(p)
	if err != nil && err != io.EOF {
		r.b.Done = true
		r.b.Status = "failed"
		r.haserr = true
	}
	r.fragment.Processed += int64(n)
	r.b.mp.haschange = true
	return n, err
}

func (r *barw) Close() error {
	if r.haserr {
		r.b.mu.Lock()
		defer r.b.mu.Unlock()
		delete(r.b.Fragments, r.fragment.uid)
	}
	return r.wc.Close()
}

type barwa struct {
	*barw
}

func (r barwa) WriteAt(p []byte, off int64) (int, error) {
	wat, ok := r.wc.(io.WriterAt)
	if !ok {
		return 0, io.ErrUnexpectedEOF
	}
	n, err := wat.WriteAt(p, off)
	if err != nil {
		r.b.Done = true
		r.b.Status = "failed"
		r.b.mp.haschange = true
		return n, err
	}
	r.fragment.Processed += int64(n)
	r.b.mp.haschange = true
	return n, nil
}
