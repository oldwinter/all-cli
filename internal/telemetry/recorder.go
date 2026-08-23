package telemetry

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
}

type traceContextKey struct{}

var traceParentPattern = regexp.MustCompile(`(?i)^[0-9a-f]{2}-([0-9a-f]{32})-[0-9a-f]{16}-[0-9a-f]{2}$`)

func New(config Config) (*Recorder, error) {
	client := config.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Second}
	}
	recorder := &Recorder{
		config:     config,
		httpClient: client,
	}

	var err error
	if strings.TrimSpace(config.SentryDSN) != "" {
		recorder.sentry, err = newSentrySink(config, client)
		if err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(config.PostHogKey) != "" {
		recorder.posthog, err = newPosthogSink(config, client)
		if err != nil {
			return nil, err
		}
	}
	return recorder, nil
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

	r.writeStructuredLog(ctx, span, command, result, duration)
	if r.config.MetricsPath != "" {
		_ = recordPrometheus(r.config.MetricsPath, command, result, duration)
	}
	if r.sentry != nil && commandErr != nil {
		r.sentry.capture(ctx, span, command, commandErr)
	}
	if r.posthog != nil {
		r.posthog.capture(ctx, command, result)
	}
}

func TraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	traceID, _ := ctx.Value(traceContextKey{}).(string)
	return traceID
}

func (r *Recorder) writeStructuredLog(ctx context.Context, span Span, command, result string, duration time.Duration) {
	if strings.TrimSpace(r.config.LogPath) == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(r.config.LogPath), 0o700); err != nil {
		return
	}
	file, err := os.OpenFile(r.config.LogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer file.Close()

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
