package telemetry

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type trackingBody struct {
	reader *strings.Reader
	reads  int
	closed bool
}

func (b *trackingBody) Read(p []byte) (int, error) {
	b.reads++
	return b.reader.Read(p)
}

func (b *trackingBody) Close() error {
	b.closed = true
	return nil
}

func TestNewPosthogSinkRejectsNonHTTPHost(t *testing.T) {
	_, err := newPosthogSink(Config{
		PostHogHost:        "ftp://analytics.example.com",
		InstallationIDPath: filepath.Join(t.TempDir(), "installation-id"),
	}, http.DefaultClient)
	if err == nil {
		t.Fatal("newPosthogSink() accepted a non-HTTP endpoint")
	}
}

func TestPosthogCaptureDrainsResponseBody(t *testing.T) {
	body := &trackingBody{reader: strings.NewReader("response")}
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: body, Header: make(http.Header)}, nil
	})}
	sink := &posthogSink{
		apiKey:         "test",
		endpoint:       "https://analytics.example.com/capture/",
		installationID: "0123456789abcdef0123456789abcdef",
		client:         client,
	}

	sink.capture(context.Background(), "status", "success")

	if body.reads == 0 || !body.closed {
		t.Fatalf("response body reads=%d closed=%v, want drained and closed", body.reads, body.closed)
	}
}
