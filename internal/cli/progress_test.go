package cli

import (
	"bytes"
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
