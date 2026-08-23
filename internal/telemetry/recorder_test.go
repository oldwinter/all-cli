package telemetry

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestRecorderWritesStructuredLifecycleAndPrometheusMetrics(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "events.jsonl")
	metricsPath := filepath.Join(root, "all-cli.prom")
	recorder, err := New(Config{
		LogPath:     logPath,
		MetricsPath: metricsPath,
		Release:     "v1.2.3",
		TraceParent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx, span := recorder.Start(context.Background())
	if span.TraceID != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("Start() trace ID = %q", span.TraceID)
	}
	if got := TraceID(ctx); got != span.TraceID {
		t.Fatalf("TraceID(ctx) = %q, want %q", got, span.TraceID)
	}
	recorder.Finish(ctx, span, "status", nil)

	event := readJSONEvent(t, logPath)
	assertEventValue(t, event, "event", "command_finished")
	assertEventValue(t, event, "command", "status")
	assertEventValue(t, event, "result", "success")
	assertEventValue(t, event, "release", "v1.2.3")
	assertEventValue(t, event, "trace_id", span.TraceID)
	if _, ok := event["duration_ms"].(float64); !ok {
		t.Fatalf("duration_ms = %#v, want JSON number", event["duration_ms"])
	}
	for _, forbidden := range []string{"args", "arguments", "environment", "path", "username", "output"} {
		if _, ok := event[forbidden]; ok {
			t.Fatalf("structured event contains forbidden field %q: %#v", forbidden, event)
		}
	}

	metrics, err := os.ReadFile(metricsPath)
	if err != nil {
		t.Fatalf("ReadFile(metrics) error = %v", err)
	}
	text := string(metrics)
	for _, want := range []string{
		`all_cli_command_total{command="status",result="success"} 1`,
		`all_cli_command_duration_seconds_count{command="status",result="success"} 1`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("metrics missing %q:\n%s", want, text)
		}
	}
	for _, forbidden := range []string{span.TraceID, "error_message", "argument"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("metrics contain high-cardinality/private value %q:\n%s", forbidden, text)
		}
	}
}

func TestRecorderGeneratesTraceAndDoesNotLogErrorDetails(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "events.jsonl")
	recorder, err := New(Config{LogPath: logPath})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx, span := recorder.Start(context.Background())
	if ok, _ := regexp.MatchString(`^[0-9a-f]{32}$`, span.TraceID); !ok {
		t.Fatalf("generated trace ID = %q, want 32 lowercase hex characters", span.TraceID)
	}
	privateError := errors.New("token=secret-value failed under /Users/private/project")
	recorder.Finish(ctx, span, "doctor", privateError)

	event := readJSONEvent(t, logPath)
	assertEventValue(t, event, "result", "error")
	encoded, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal(event) error = %v", err)
	}
	for _, forbidden := range []string{"secret-value", "/Users/private", privateError.Error()} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("structured log leaked %q: %s", forbidden, encoded)
		}
	}
}

func TestPrometheusMetricsAggregateAcrossRecorderInstances(t *testing.T) {
	metricsPath := filepath.Join(t.TempDir(), "all-cli.prom")
	for _, resultErr := range []error{nil, errors.New("failed")} {
		recorder, err := New(Config{MetricsPath: metricsPath})
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		ctx, span := recorder.Start(context.Background())
		span.StartedAt = time.Now().Add(-250 * time.Millisecond)
		recorder.Finish(ctx, span, "version", resultErr)
	}

	metrics, err := os.ReadFile(metricsPath)
	if err != nil {
		t.Fatalf("ReadFile(metrics) error = %v", err)
	}
	text := string(metrics)
	for _, want := range []string{
		`all_cli_command_total{command="version",result="success"} 1`,
		`all_cli_command_total{command="version",result="error"} 1`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("metrics missing %q:\n%s", want, text)
		}
	}
}

func readJSONEvent(t *testing.T, path string) map[string]any {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open(%q) error = %v", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		t.Fatalf("no JSON event in %s: %v", path, scanner.Err())
	}
	var event map[string]any
	if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
		t.Fatalf("Unmarshal(event) error = %v\n%s", err, scanner.Bytes())
	}
	return event
}

func assertEventValue(t *testing.T, event map[string]any, key string, want any) {
	t.Helper()
	if got := event[key]; got != want {
		t.Fatalf("event[%q] = %#v, want %#v; event=%#v", key, got, want, event)
	}
}
