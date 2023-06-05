package progress

import (
	"fmt"
	"io"
	"math"
	"strings"
	"sync"

	"kubegems.io/modelx/pkg/client/units"
)

var SpinnerDefault = []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}

type Bar struct {
	Name       string
	MaxNameLen int    // max name length
	Total      int64  // total bytes, -1 for indeterminate
	Width      int    // width of the bar
	Status     string // status text
	Done       bool   // if the bar is done
	Fragments  map[string]*BarFragment

	nameindex    int // scroll name index
	refreshcount int // refresh count for scroll name
	mu           sync.Mutex
	mp           *MultiBar
}

type BarFragment struct {
	Offset       int64  // offset of the fragment
	Processed    int64  // processed bytes
	uid          string // uid of the fragment, for delete
	nototalindex int    // index when no total
}

func (b *Bar) SetNameStatus(name, status string, done bool) {
	b.Name, b.Status, b.Done = name, status, done
	b.Notify()
}

func (b *Bar) SetStatus(status string, done bool) {
	b.Status = status
	b.Done = done
	b.Notify()
}

func (b *Bar) SetDone() {
	b.Done = true
	b.Notify()
}

func (r *Bar) Notify() {
	if r.mp != nil {
		r.mp.print()
	}
}

func (b *Bar) Print(w io.Writer) {
	processwidth := b.Width

	buff := make([]byte, processwidth)
	status := ""
	if b.Done {
		for i := range buff {
			buff[i] = '+'
		}
		status = b.Status
	} else {
		for i := range buff {
			buff[i] = '-'
		}
		var totalProcessed int64
		for _, f := range b.Fragments {
			totalProcessed += f.Processed
			if b.Total > 0 {
				start := int(float64(processwidth) * float64(f.Offset) / float64(b.Total))
				end := int(float64(processwidth) * float64(f.Offset+f.Processed) / float64(b.Total))
				if end > processwidth {
					end = processwidth
				}
				if start < 0 {
					start = 0
				}
				for i := start; i < end; i++ {
					buff[i] = '+'
				}
			} else {
				buff[f.nototalindex%processwidth] = '+'
				f.nototalindex++
			}
		}
		if totalProcessed > 0 {
			if b.Total <= 0 {
				status = units.HumanSize(float64(totalProcessed))
			} else {
				status = units.HumanSize(float64(totalProcessed)) + "/" + units.HumanSize(float64(b.Total))
			}
		} else {
			status = b.Status
		}
	}
	showname := b.Name
	if len(b.Name) > b.MaxNameLen {
		b.mp.haschange = true // force print
		fullname := b.Name + "  "
		lowptr := b.nameindex % len(fullname)
		maxptr := lowptr + b.MaxNameLen
		if maxptr < len(fullname) {
			showname = fullname[lowptr:maxptr]
		} else {
			showname = fullname[lowptr:] + fullname[:maxptr-len(fullname)]
		}
		// 3x speed low than fps
		if b.refreshcount%3 == 0 {
			b.nameindex++
		}
		b.refreshcount++
	} else if len(showname) < b.MaxNameLen {
		showname += strings.Repeat(" ", b.MaxNameLen-len(showname))
	}
	fmt.Fprintf(w, "%s [%s] %s\n", showname, string(buff), status)
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
