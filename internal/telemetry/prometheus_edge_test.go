package telemetry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRecordPrometheusIgnoresMalformedSamples(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.prom")
	input := strings.Join([]string{
		`all_cli_command_total{command="status",result="success"} -1`,
		`all_cli_command_duration_seconds_sum{command="status",result="success"} NaN`,
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := recordPrometheus(path, "status", "success", time.Second); err != nil {
		t.Fatalf("recordPrometheus() error = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	if !strings.Contains(text, `all_cli_command_total{command="status",result="success"} 1`) ||
		!strings.Contains(text, `all_cli_command_duration_seconds_sum{command="status",result="success"} 1.000000`) {
		t.Fatalf("malformed samples poisoned rewritten metrics:\n%s", text)
	}
}

func TestRecordPrometheusDoesNotMaterializeForeignMetrics(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.prom")
	input := `foreign_metric{command="ghost",result="success"} 99` + "\n"
	if err := os.WriteFile(path, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := recordPrometheus(path, "status", "success", time.Second); err != nil {
		t.Fatalf("recordPrometheus() error = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), `command="ghost"`) {
		t.Fatalf("foreign metric created official zero-value series:\n%s", got)
	}
}

func TestRecordPrometheusPreservesLargeIntegerCounters(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.prom")
	input := `all_cli_command_total{command="status",result="success"} 9007199254740993` + "\n"
	if err := os.WriteFile(path, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := recordPrometheus(path, "status", "success", time.Second); err != nil {
		t.Fatalf("recordPrometheus() error = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), `all_cli_command_total{command="status",result="success"} 9007199254740994`) {
		t.Fatalf("large counter lost integer precision:\n%s", got)
	}
}
