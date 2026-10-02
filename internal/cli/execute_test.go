package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteLeavesTelemetryDisabledByDefault(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "events.jsonl")
	env := map[string]string{
		"ALL_CLI_LOG_PATH": logPath,
	}
	var stdout, stderr bytes.Buffer

	err := Execute(context.Background(), []string{"version"}, &stdout, &stderr, mapGetenv(env))

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if stdout.Len() == 0 {
		t.Fatal("Execute(version) produced no stdout")
	}
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatalf("telemetry artifact exists without feature flag: %v", err)
	}
}

func TestExecuteRecordsEnabledCommandLifecycle(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "events.jsonl")
	metricsPath := filepath.Join(root, "all-cli.prom")
	env := map[string]string{
		"ALL_CLI_FEATURES":     "telemetry-v1",
		"ALL_CLI_LOG_PATH":     logPath,
		"ALL_CLI_METRICS_PATH": metricsPath,
	}
	var stdout, stderr bytes.Buffer

	err := Execute(context.Background(), []string{"version"}, &stdout, &stderr, mapGetenv(env))

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	event := readExecuteEvent(t, logPath)
	if event["command"] != "version" || event["result"] != "success" {
		t.Fatalf("telemetry event = %#v", event)
	}
	if _, err := os.Stat(metricsPath); err != nil {
		t.Fatalf("metrics file missing: %v", err)
	}
}

func TestExecuteRecordsInvalidInputAsUnknownCommand(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "events.jsonl")
	privateInput := "token=do-not-record"
	env := map[string]string{
		"ALL_CLI_FEATURES": "telemetry-v1",
		"ALL_CLI_LOG_PATH": logPath,
	}
	var stdout, stderr bytes.Buffer

	err := Execute(context.Background(), []string{privateInput}, &stdout, &stderr, mapGetenv(env))

	if err == nil {
		t.Fatal("Execute(invalid command) error = nil")
	}
	event := readExecuteEvent(t, logPath)
	if event["command"] != "unknown" || event["result"] != "error" {
		t.Fatalf("telemetry event = %#v", event)
	}
	encoded, _ := json.Marshal(event)
	if strings.Contains(string(encoded), privateInput) {
		t.Fatalf("telemetry leaked invalid user input: %s", encoded)
	}
}

func TestExecuteRejectsUnknownFeatureFlags(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := Execute(context.Background(), []string{"version"}, &stdout, &stderr, mapGetenv(map[string]string{
		"ALL_CLI_FEATURES": "future-mode",
	}))

	if err == nil || !strings.Contains(err.Error(), "unknown ALL_CLI_FEATURES") {
		t.Fatalf("Execute() error = %v, want unknown feature error", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Error: unknown ALL_CLI_FEATURES values: future-mode") {
		t.Fatalf("stderr = %q, want actionable unknown feature error", stderr.String())
	}
}

func mapGetenv(values map[string]string) func(string) string {
	return func(key string) string {
		return values[key]
	}
}

func readExecuteEvent(t *testing.T, path string) map[string]any {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	line := strings.SplitN(string(content), "\n", 2)[0]
	var event map[string]any
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		t.Fatalf("Unmarshal(event) error = %v\n%s", err, line)
	}
	return event
}

func TestExecuteToleratesNilDeps(t *testing.T) {
	var stdout, stderr bytes.Buffer
	var nilCtx context.Context
	if err := Execute(nilCtx, []string{"version"}, &stdout, &stderr, nil); err != nil {
		t.Fatalf("Execute(nil deps) error = %v", err)
	}
	if stdout.Len() == 0 {
		t.Fatal("Execute(nil deps) produced no stdout")
	}
}

func TestExecuteReportsTelemetryInitFailure(t *testing.T) {
	env := map[string]string{
		"ALL_CLI_FEATURES": "telemetry-v1",
		"SENTRY_DSN":       "not-a-dsn",
	}
	var stdout, stderr bytes.Buffer
	err := Execute(context.Background(), []string{"version"}, &stdout, &stderr, mapGetenv(env))
	if err == nil || !strings.Contains(err.Error(), "initialize telemetry") {
		t.Fatalf("Execute() error = %v, want telemetry init failure", err)
	}
	if !strings.Contains(stderr.String(), "initialize telemetry") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestTelemetryCommandNaming(t *testing.T) {
	root := NewRootCommand()
	if got := telemetryCommand(root, []string{"--version"}); got != "version" {
		t.Fatalf("--version => %q", got)
	}
	if got := telemetryCommand(root, []string{"status"}); got != "status" {
		t.Fatalf("status => %q", got)
	}
	if got := telemetryCommand(root, []string{"bogus"}); got != "unknown" {
		t.Fatalf("bogus => %q", got)
	}
}
