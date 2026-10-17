package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

// captureServer records every request body sent to a fake Sentry/PostHog endpoint.
type captureServer struct {
	mu     sync.Mutex
	bodies []string
	server *httptest.Server
}

func newCaptureServer(t *testing.T) *captureServer {
	t.Helper()
	c := &captureServer{}
	c.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		c.mu.Lock()
		c.bodies = append(c.bodies, string(body))
		c.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(c.server.Close)
	return c
}

func (c *captureServer) payloads() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.bodies...)
}

// sentryFixture returns a Config pointing at a local sink.
func sentryFixture(t *testing.T, sinkURL string) Config {
	t.Helper()
	// Sentry DSN form: scheme://publickey@host/projectID. The sink URL already
	// carries scheme://host:port, so insert the public key before the host.
	dsn := strings.Replace(sinkURL, "://", "://publickey@", 1) + "/1"
	return Config{
		SentryDSN:          dsn,
		SentryEnvironment:  "test",
		Release:            "test-release",
		InstallationIDPath: t.TempDir() + "/installation-id",
	}
}

// TestSentryNeverTransmitsSensitiveContent is the end-to-end regression test for
// the privacy contract in docs/observability.md. Each fixture was captured from
// a real code path that previously reached the network verbatim.
//
// Every credential below is a deliberately low-entropy placeholder so secret
// scanners do not mistake this test data for a live credential. The shapes still
// exercise the same scrubber paths as the real leaks they stand in for.
func TestSentryNeverTransmitsSensitiveContent(t *testing.T) {
	tests := []struct {
		name        string
		commandErr  error
		mustNotHold []string
	}{
		{
			name:        "bearer token",
			commandErr:  errors.New("request failed: Authorization: Bearer placeholder-bearer-value-not-a-real-secret"),
			mustNotHold: []string{"placeholder-bearer-value-not-a-real-secret"},
		},
		{
			name:        "json form secret",
			commandErr:  errors.New(`decode config {"password":"hunter2","user":"alice"}`),
			mustNotHold: []string{"hunter2"},
		},
		{
			name:        "aws secret access key",
			commandErr:  errors.New("aws failed AWS_SECRET_ACCESS_KEY=placeholder-aws-value-not-a-real-secret"),
			mustNotHold: []string{"placeholder-aws-value-not-a-real-secret"},
		},
		{
			name:        "url userinfo credentials",
			commandErr:  errors.New("clone failed https://alice:hunter2@gitlab.example.com/api/v4"),
			mustNotHold: []string{"alice", "hunter2"},
		},
		{
			name:        "private hostname",
			commandErr:  errors.New("gh auth switch failed (exit=1): not logged in to github.acme-corp.internal"),
			mustNotHold: []string{"acme-corp.internal"},
		},
		{
			name:        "windows path with username",
			commandErr:  errors.New(`read snapshot C:\Users\John Doe\private\before.json: open failed`),
			mustNotHold: []string{"John", "Doe", "private", `C:\Users`},
		},
		{
			name:        "posix home path",
			commandErr:  errors.New("failed to read /home/alice/.aws/credentials"),
			mustNotHold: []string{"alice", ".aws"},
		},
		{
			name:        "kubectl context name",
			commandErr:  errors.New(`kubectl config use-context "acme-internal-staging" failed (exit=1)`),
			mustNotHold: []string{"acme-internal-staging"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink := newCaptureServer(t)
			config := sentryFixture(t, sink.server.URL)
			recorder, err := New(config)
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			ctx, span := recorder.Start(context.Background())
			recorder.Finish(ctx, span, "docker use", tt.commandErr)

			payloads := sink.payloads()
			if len(payloads) == 0 {
				t.Fatal("expected a Sentry payload to be delivered")
			}
			wire := strings.Join(payloads, "\n")
			for _, secret := range tt.mustNotHold {
				if strings.Contains(wire, secret) {
					t.Fatalf("Sentry payload leaked %q\npayload: %s", secret, wire)
				}
			}
		})
	}
}

// TestSentryPayloadCarriesUsefulClassification ensures the privacy fix did not
// strip the signal operators actually need.
func TestSentryPayloadCarriesUsefulClassification(t *testing.T) {
	sink := newCaptureServer(t)
	recorder, err := New(sentryFixture(t, sink.server.URL))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx, span := recorder.Start(context.Background())
	recorder.Finish(ctx, span, "docker use", context.DeadlineExceeded)

	payloads := sink.payloads()
	if len(payloads) == 0 {
		t.Fatal("expected a payload")
	}
	wire := strings.Join(payloads, "\n")
	if !strings.Contains(wire, "timeout") {
		t.Fatalf("expected the timeout classification in the payload, got %s", wire)
	}
	if !strings.Contains(wire, `"transaction":"docker use"`) {
		t.Fatalf("expected the normalized command path, got %s", wire)
	}
}

func TestErrorClass(t *testing.T) {
	exitErr := &exec.ExitError{}
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"nil", nil, "none"},
		{"deadline", context.DeadlineExceeded, "timeout"},
		{"canceled", context.Canceled, "canceled"},
		{"exit error", exitErr, "exit_nonzero"},
		{"not found", errors.New("exec: \"docker\": executable file not found in %PATH%"), "not_found"},
		{"permission", errors.New("open file: permission denied"), "permission_denied"},
		{"usage", errors.New(`unknown flag: --fix`), "invalid_usage"},
		{"snapshot", errors.New("parse snapshot stdin: missing schema_version"), "invalid_snapshot"},
		{"network", errors.New("dial tcp: connection refused"), "network"},
		{"generic failure", errors.New("docker context use failed"), "command_failed"},
		{"fallback", errors.New("something odd"), "error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errorClass(tt.err); got != tt.want {
				t.Fatalf("errorClass(%v) = %q, want %q", tt.err, got, tt.want)
			}
		})
	}
}

func TestMarkBenignExitIsRecognised(t *testing.T) {
	base := errors.New("snapshot differences found")
	wrapped := MarkBenignExit(base)

	if !isBenignExit(wrapped) {
		t.Fatal("expected MarkBenignExit result to be recognised as benign")
	}
	if isBenignExit(base) {
		t.Fatal("expected a plain error to not be benign")
	}
	if !errors.Is(wrapped, base) {
		t.Fatal("expected the wrapped error to unwrap to the original")
	}
	if MarkBenignExit(nil) != nil {
		t.Fatal("expected MarkBenignExit(nil) to be nil")
	}
}

// TestBenignExitIsNotSentToSentry verifies that `diff --exit-code` reporting a
// difference is not treated as a crash.
func TestBenignExitIsNotSentToSentry(t *testing.T) {
	sink := newCaptureServer(t)
	recorder, err := New(sentryFixture(t, sink.server.URL))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx, span := recorder.Start(context.Background())
	recorder.Finish(ctx, span, "diff", MarkBenignExit(errors.New("snapshot differences found")))

	if payloads := sink.payloads(); len(payloads) != 0 {
		t.Fatalf("expected no Sentry payload for a benign exit, got %v", payloads)
	}
}

// TestNewDegradesInsteadOfFailing verifies that a misconfigured sink disables
// only that sink. Previously a typo'd POSTHOG_HOST made the whole CLI exit 1.
func TestNewDegradesInsteadOfFailing(t *testing.T) {
	recorder, err := New(Config{
		PostHogKey:  "phc_test",
		PostHogHost: "not a url",
	})
	if err != nil {
		t.Fatalf("expected no error from a bad PostHog host, got %v", err)
	}
	if recorder == nil {
		t.Fatal("expected a recorder")
	}
	if recorder.posthog != nil {
		t.Fatal("expected the posthog sink to be disabled")
	}
	warnings := recorder.Warnings()
	if len(warnings) == 0 {
		t.Fatal("expected a warning describing the disabled sink")
	}
	if !strings.Contains(strings.Join(warnings, " "), "posthog") {
		t.Fatalf("expected the warning to name posthog, got %v", warnings)
	}
}

func TestNewKeepsValidSinks(t *testing.T) {
	recorder, err := New(Config{
		PostHogKey:         "phc_test",
		PostHogHost:        "https://us.i.posthog.com",
		InstallationIDPath: t.TempDir() + "/installation-id",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if recorder.posthog == nil {
		t.Fatal("expected the posthog sink to be constructed")
	}
	if len(recorder.Warnings()) != 0 {
		t.Fatalf("expected no warnings, got %v", recorder.Warnings())
	}
}

func TestWarningsOnNilRecorder(t *testing.T) {
	var recorder *Recorder
	if got := recorder.Warnings(); got != nil {
		t.Fatalf("expected nil warnings from a nil recorder, got %v", got)
	}
}

// TestFinishDoesNotBlockOnSlowSink verifies the shared 200ms exit budget.
func TestFinishDoesNotBlockOnSlowSink(t *testing.T) {
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(slow.Close)

	recorder, err := New(sentryFixture(t, slow.URL))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx, span := recorder.Start(context.Background())

	start := time.Now()
	recorder.Finish(ctx, span, "version", errors.New("boom"))
	elapsed := time.Since(start)

	if elapsed > time.Second {
		t.Fatalf("Finish blocked for %v; the exit budget is %v", elapsed, finishBudget)
	}
}

// TestPostHogPayloadHasNoErrorText verifies PostHog receives only the documented
// anonymous fields.
func TestPostHogPayloadHasNoErrorText(t *testing.T) {
	sink := newCaptureServer(t)
	recorder, err := New(Config{
		PostHogKey:         "phc_test",
		PostHogHost:        sink.server.URL,
		InstallationIDPath: t.TempDir() + "/installation-id",
		Release:            "test-release",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx, span := recorder.Start(context.Background())
	recorder.Finish(ctx, span, "docker use", errors.New("secret-context-name failed"))

	payloads := sink.payloads()
	if len(payloads) == 0 {
		t.Fatal("expected a PostHog payload")
	}
	wire := strings.Join(payloads, "\n")
	if strings.Contains(wire, "secret-context-name") {
		t.Fatalf("PostHog payload leaked the error text: %s", wire)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(payloads[0]), &decoded); err != nil {
		t.Fatalf("payload is not valid JSON: %v", err)
	}
	props, ok := decoded["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties in payload, got %v", decoded)
	}
	if props["command"] != "docker use" {
		t.Fatalf("command = %v, want %q", props["command"], "docker use")
	}
	if props["result"] != "error" {
		t.Fatalf("result = %v, want %q", props["result"], "error")
	}
}

// TestFinishDeliversWhenParentContextCancelled guards the context.WithoutCancel
// fix: a cancelled command context must not silently drop telemetry.
func TestFinishDeliversWhenParentContextCancelled(t *testing.T) {
	sink := newCaptureServer(t)
	recorder, err := New(sentryFixture(t, sink.server.URL))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	ctx, span := recorder.Start(cancelled)
	cancel()

	recorder.Finish(ctx, span, "version", errors.New("boom"))

	// Delivery is bounded by finishBudget, so poll briefly instead of assuming
	// the request has already landed when Finish returns.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(sink.payloads()) > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expected delivery even though the parent context was cancelled")
}
