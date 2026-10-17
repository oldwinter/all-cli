package telemetry

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Config struct {
	LogPath            string
	MetricsPath        string
	SentryDSN          string
	SentryEnvironment  string
	PostHogKey         string
	PostHogHost        string
	InstallationIDPath string
	Release            string
	TraceParent        string
	HTTPClient         *http.Client
}

type Span struct {
	TraceID   string
	StartedAt time.Time
}

type Recorder struct {
	config     Config
	httpClient *http.Client
	sentry     *sentrySink
	posthog    *posthogSink
	warnings   []string
}

type traceContextKey struct{}

var traceParentPattern = regexp.MustCompile(`(?i)^[0-9a-f]{2}-([0-9a-f]{32})-[0-9a-f]{16}-[0-9a-f]{2}$`)

// New builds a Recorder from config.
//
// Misconfigured individual sinks never fail the whole Recorder: a bad
// SENTRY_DSN, POSTHOG_HOST, or an unwritable installation-ID path disables that
// one sink and is reported through Warnings. This keeps the documented promise
// that telemetry problems never change a command's output or exit status.
// The only hard failure is a Config that cannot be represented at all.
func New(config Config) (*Recorder, error) {
	client := config.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Second}
	}
	recorder := &Recorder{
		config:     config,
		httpClient: client,
	}

	if strings.TrimSpace(config.SentryDSN) != "" {
		sink, err := newSentrySink(config, client)
		if err != nil {
			recorder.warnings = append(recorder.warnings, fmt.Sprintf("sentry disabled: %v", err))
		} else {
			recorder.sentry = sink
		}
	}
	if strings.TrimSpace(config.PostHogKey) != "" {
		sink, err := newPosthogSink(config, client)
		if err != nil {
			recorder.warnings = append(recorder.warnings, fmt.Sprintf("posthog disabled: %v", err))
		} else {
			recorder.posthog = sink
		}
	}
	return recorder, nil
}

// Warnings reports sinks that were disabled because they were misconfigured.
func (r *Recorder) Warnings() []string {
	if r == nil {
		return nil
	}
	return append([]string(nil), r.warnings...)
}

func (r *Recorder) Start(ctx context.Context) (context.Context, Span) {
	if ctx == nil {
		ctx = context.Background()
	}
	traceID := traceIDFromParent(r.config.TraceParent)
	if traceID == "" {
		traceID = randomHex(16)
	}
	span := Span{
		TraceID:   traceID,
		StartedAt: time.Now(),
	}
	return context.WithValue(ctx, traceContextKey{}, traceID), span
}

// finishBudget bounds the total wall time Finish may add to a command's exit.
// A CLI must never make the user wait on analytics; this is a single shared
// deadline for all remote sinks, not a per-sink timeout.
const finishBudget = 200 * time.Millisecond

// benignExitError marks an error that represents requested control flow rather
// than a failure, so it is not reported to Sentry as a crash.
type benignExitError struct{ err error }

func (b benignExitError) Error() string { return b.err.Error() }
func (b benignExitError) Unwrap() error { return b.err }

// MarkBenignExit wraps err so telemetry records it as expected control flow
// (for example `diff --exit-code` reporting that snapshots differ).
func MarkBenignExit(err error) error {
	if err == nil {
		return nil
	}
	return benignExitError{err: err}
}

// isBenignExit reports whether err was produced by intended control flow.
func isBenignExit(err error) bool {
	var benign benignExitError
	return errors.As(err, &benign)
}

func (r *Recorder) Finish(ctx context.Context, span Span, command string, commandErr error) {
	result := "success"
	if commandErr != nil {
		result = "error"
	}
	duration := time.Since(span.StartedAt)
	if duration < 0 {
		duration = 0
	}
	command = normalizeCommand(command)

	// Local sinks are fast and must not be lost, so run them synchronously.
	_ = r.writeStructuredLog(ctx, span, command, result, duration)
	if r.config.MetricsPath != "" {
		_ = recordPrometheus(r.config.MetricsPath, command, result, duration)
	}

	// Detach remote delivery from the command's context: by the time Finish
	// runs the command has completed, and a cancelled parent context would
	// otherwise silently discard every event.
	remoteCtx := context.WithoutCancel(ctx)

	var wg sync.WaitGroup
	if r.sentry != nil && commandErr != nil && !isBenignExit(commandErr) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.sentry.capture(remoteCtx, span, command, commandErr)
		}()
	}
	if r.posthog != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.posthog.capture(remoteCtx, command, result)
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(finishBudget):
		// Give up waiting. main() exits via os.Exit, so abandoned goroutines
		// die with the process; the user's command is never blocked further.
	}
}

func TraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	traceID, _ := ctx.Value(traceContextKey{}).(string)
	return traceID
}

func (r *Recorder) writeStructuredLog(ctx context.Context, span Span, command, result string, duration time.Duration) error {
	if strings.TrimSpace(r.config.LogPath) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(r.config.LogPath), 0o700); err != nil {
		return fmt.Errorf("create structured log directory: %w", err)
	}
	file, err := os.OpenFile(r.config.LogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open structured log: %w", err)
	}

	logger := slog.New(slog.NewJSONHandler(file, &slog.HandlerOptions{Level: slog.LevelInfo}))
	logger.InfoContext(
		ctx,
		"command finished",
		slog.String("event", "command_finished"),
		slog.String("command", command),
		slog.String("result", result),
		slog.Int64("duration_ms", duration.Milliseconds()),
		slog.String("release", r.config.Release),
		slog.String("trace_id", span.TraceID),
	)
	if err := file.Close(); err != nil {
		return fmt.Errorf("close structured log: %w", err)
	}
	return nil
}

func traceIDFromParent(traceParent string) string {
	match := traceParentPattern.FindStringSubmatch(strings.TrimSpace(traceParent))
	if len(match) != 2 || match[1] == strings.Repeat("0", 32) {
		return ""
	}
	return strings.ToLower(match[1])
}

func randomHex(bytesCount int) string {
	data := make([]byte, bytesCount)
	if _, err := rand.Read(data); err != nil {
		now := time.Now().UTC().Format(time.RFC3339Nano)
		return hex.EncodeToString([]byte(now))[:bytesCount*2]
	}
	return hex.EncodeToString(data)
}

func normalizeCommand(command string) string {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return "unknown"
	}
	return strings.Join(fields, " ")
}
