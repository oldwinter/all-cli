package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestRecorderSendsContextualizedSentryAndMinimizedPostHogEvents(t *testing.T) {
	var mu sync.Mutex
	requests := make(map[string][]byte)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("ReadAll(request) error = %v", err)
			return
		}
		mu.Lock()
		requests[r.URL.Path] = body
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	installationIDPath := filepath.Join(t.TempDir(), "installation-id")
	sentryDSN := strings.Replace(server.URL, "http://", "http://public@", 1) + "/42"
	recorder, err := New(Config{
		SentryDSN:          sentryDSN,
		SentryEnvironment:  "ci",
		PostHogKey:         "phc_test",
		PostHogHost:        server.URL,
		InstallationIDPath: installationIDPath,
		Release:            "v1.2.3",
		HTTPClient:         server.Client(),
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx, span := recorder.Start(context.Background())
	recorder.Finish(ctx, span, "doctor", errors.New("token=top-secret failed at /Users/private/config"))

	mu.Lock()
	sentryEnvelope := append([]byte(nil), requests["/api/42/envelope/"]...)
	posthogPayload := append([]byte(nil), requests["/capture/"]...)
	mu.Unlock()
	if len(sentryEnvelope) == 0 {
		t.Fatal("Sentry envelope was not sent")
	}
	if !bytes.Contains(sentryEnvelope, []byte(`"transaction":"doctor"`)) ||
		!bytes.Contains(sentryEnvelope, []byte(`"environment":"ci"`)) ||
		!bytes.Contains(sentryEnvelope, []byte(`"release":"v1.2.3"`)) ||
		!bytes.Contains(sentryEnvelope, []byte(span.TraceID)) ||
		!bytes.Contains(sentryEnvelope, []byte(`"breadcrumbs"`)) ||
		!bytes.Contains(sentryEnvelope, []byte(`"stacktrace"`)) {
		t.Fatalf("Sentry envelope lacks context:\n%s", sentryEnvelope)
	}
	for _, forbidden := range []string{"top-secret", "/Users/private"} {
		if bytes.Contains(sentryEnvelope, []byte(forbidden)) {
			t.Fatalf("Sentry envelope leaked %q:\n%s", forbidden, sentryEnvelope)
		}
	}

	var capture struct {
		APIKey     string         `json:"api_key"`
		Event      string         `json:"event"`
		Properties map[string]any `json:"properties"`
	}
	if err := json.Unmarshal(posthogPayload, &capture); err != nil {
		t.Fatalf("Unmarshal(PostHog payload) error = %v\n%s", err, posthogPayload)
	}
	if capture.APIKey != "phc_test" || capture.Event != "all_cli command" {
		t.Fatalf("PostHog capture = %#v", capture)
	}
	for key, want := range map[string]any{
		"command": "doctor",
		"result":  "error",
		"release": "v1.2.3",
	} {
		if got := capture.Properties[key]; got != want {
			t.Fatalf("PostHog properties[%q] = %#v, want %#v", key, got, want)
		}
	}
	if distinctID, _ := capture.Properties["distinct_id"].(string); distinctID == "" {
		t.Fatalf("PostHog distinct_id is empty: %#v", capture.Properties)
	}
	for _, forbidden := range []string{"error", "trace_id", "args", "path", "environment", "top-secret"} {
		if _, ok := capture.Properties[forbidden]; ok {
			t.Fatalf("PostHog properties contain forbidden key %q: %#v", forbidden, capture.Properties)
		}
	}
}

func TestSinkFailureDoesNotBlockOtherSinks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	root := t.TempDir()
	logPath := filepath.Join(root, "events.jsonl")
	metricsPath := filepath.Join(root, "all-cli.prom")
	recorder, err := New(Config{
		LogPath:     logPath,
		MetricsPath: metricsPath,
		SentryDSN:   strings.Replace(server.URL, "http://", "http://public@", 1) + "/42",
		HTTPClient:  server.Client(),
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx, span := recorder.Start(context.Background())
	recorder.Finish(ctx, span, "status", errors.New("failed"))

	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("structured log was blocked by Sentry failure: %v", err)
	}
	if _, err := os.Stat(metricsPath); err != nil {
		t.Fatalf("metrics were blocked by Sentry failure: %v", err)
	}
}
