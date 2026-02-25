package cli

import (
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type progressSpinner struct {
	w     io.Writer
	total int32

	done atomic.Int32
	last atomic.Value // string

	stopOnce sync.Once
	stopCh   chan struct{}

	stoppedCh chan struct{}
}

func newProgressSpinner(w io.Writer, total int) *progressSpinner {
	p := &progressSpinner{
		w:      w,
		total:  int32(total),
		stopCh: make(chan struct{}),
	}
	p.last.Store("")
	return p
}

func (p *progressSpinner) Start() {
	p.stoppedCh = make(chan struct{})
	go func() {
		defer close(p.stoppedCh)

		frames := []string{"|", "/", "-", "\\"}
		ticker := time.NewTicker(120 * time.Millisecond)
		defer ticker.Stop()

		i := 0
		for {
			select {
			case <-p.stopCh:
				fmt.Fprint(p.w, "\r\033[K")
				return
			case <-ticker.C:
				done := p.done.Load()
				last, _ := p.last.Load().(string)

				msg := fmt.Sprintf("\r\033[K%s Checking tools... %d/%d", frames[i%len(frames)], done, p.total)
				if last != "" {
					msg += fmt.Sprintf(" (last: %s)", last)
				}
				fmt.Fprint(p.w, msg)
				i++
			}
		}
	}()
}

func (p *progressSpinner) Inc(lastToolID string) {
	if lastToolID != "" {
		p.last.Store(lastToolID)
	}
	p.done.Add(1)
}

func (p *progressSpinner) Stop() {
	p.stopOnce.Do(func() { close(p.stopCh) })
	if p.stoppedCh != nil {
		<-p.stoppedCh
	}
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
