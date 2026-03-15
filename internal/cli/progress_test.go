package cli

import (
	"bytes"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

type blockingWriter struct {
	started chan struct{}
	unblock chan struct{}

	mu  sync.Mutex
	buf bytes.Buffer
}

func newBlockingWriter() *blockingWriter {
	return &blockingWriter{
		started: make(chan struct{}),
		unblock: make(chan struct{}),
	}
}

func (w *blockingWriter) Write(p []byte) (int, error) {
	select {
	case <-w.started:
		// already closed
	default:
		close(w.started)
	}

	<-w.unblock

	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.Write(p)
}

func (w *blockingWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

type captureWriter struct {
	ch chan string
}

func newCaptureWriter() *captureWriter {
	return &captureWriter{ch: make(chan string, 64)}
}

func (w *captureWriter) Write(p []byte) (int, error) {
	w.ch <- string(p)
	return len(p), nil
}

func TestProgressSpinner_StopBlocksUntilGoroutineExits(t *testing.T) {
	w := newBlockingWriter()
	p := newProgressSpinner(w, 1)
	p.Start()

	select {
	case <-w.started:
		// spinner attempted first write and is blocked in Write.
	case <-time.After(2 * time.Second):
		t.Fatal("spinner did not write within timeout")
	}

	stopped := make(chan struct{})
	go func() {
		p.Stop()
		close(stopped)
	}()

	select {
	case <-stopped:
		t.Fatal("Stop returned while spinner goroutine was still blocked writing")
	case <-time.After(100 * time.Millisecond):
		// expected: Stop should wait for the goroutine to exit.
	}

	close(w.unblock)

	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop did not return within timeout after unblocking writes")
	}

	if out := w.String(); !strings.Contains(out, "\r\033[K") {
		t.Fatalf("expected clear sequence in output, got %q", out)
	}
}

func TestProgressSpinner_ClearsLineOnEachUpdate(t *testing.T) {
	w := newCaptureWriter()
	p := newProgressSpinner(w, 2)
	p.Start()
	defer p.Stop()

	var first string
	select {
	case first = <-w.ch:
	case <-time.After(2 * time.Second):
		t.Fatal("spinner did not write within timeout")
	}

	if !strings.HasPrefix(first, "\r\033[K") {
		t.Fatalf("expected spinner update to clear line (\\r\\033[K prefix), got %q", first)
	}
}

func TestProgressSpinner_ShowsLastTool(t *testing.T) {
	w := newCaptureWriter()
	p := newProgressSpinner(w, 2)
	p.Inc("kubectl")
	p.Start()
	defer p.Stop()

	var first string
	select {
	case first = <-w.ch:
	case <-time.After(2 * time.Second):
		t.Fatal("spinner did not write within timeout")
	}

	if !strings.Contains(first, "(last: kubectl)") {
		t.Fatalf("expected last tool in spinner output, got %q", first)
	}
}

func TestProgressSpinner_IncUpdatesCounters(t *testing.T) {
	p := newProgressSpinner(&bytes.Buffer{}, 3)

	p.Inc("kubectl")
	p.Inc("")

	if got := p.done.Load(); got != 2 {
		t.Fatalf("done = %d, want 2", got)
	}
	last, _ := p.last.Load().(string)
	if last != "kubectl" {
		t.Fatalf("last = %q, want %q", last, "kubectl")
	}
}

func TestIsTerminalFalseOnClosedFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "closed")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	name := f.Name()
	if err := f.Close(); err != nil {
		t.Fatalf("close temp: %v", err)
	}
	_ = name

	if isTerminal(f) {
		t.Fatal("expected closed file to be non-terminal")
	}
}

func TestIsTerminalFalseOnRegularFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "regular")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	defer f.Close()

	if isTerminal(f) {
		t.Fatal("expected regular file to be non-terminal")
	}
}

func TestIsTerminalTrueOnDevNull(t *testing.T) {
	f, err := os.Open("/dev/null")
	if err != nil {
		t.Fatalf("open /dev/null: %v", err)
	}
	defer f.Close()

	if !isTerminal(f) {
		t.Fatal("expected /dev/null to be treated as a character device")
	}
}
