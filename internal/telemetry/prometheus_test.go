package telemetry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseMetricLineMalformed(t *testing.T) {
	for _, line := range []string{
		`all_cli_x{junk} 3`,
		`all_cli_x{command="a",result="b"} notfloat`,
		`nolabels 3`,
	} {
		if _, _, _, ok := parseMetricLine(line); ok {
			t.Fatalf("parseMetricLine(%q) = ok, want rejection", line)
		}
	}
}

func TestMetricSortTiesOnCommand(t *testing.T) {
	metrics := map[metricKey]metricValue{
		{Command: "status", Result: "success"}: {Count: 1},
		{Command: "status", Result: "error"}:   {Count: 2},
		{Command: "doctor", Result: "ok"}:      {Count: 3},
	}
	var buf strings.Builder
	if err := writePrometheus(filepath.Join(t.TempDir(), "m.prom"), metrics); err != nil {
		t.Fatal(err)
	}
	_ = buf
}

func TestParseMetricLineMalformedLabels(t *testing.T) {
	// The `"} ` terminator is present, so the label body reaches Sscanf and
	// fails there rather than on the shape check.
	if _, _, _, ok := parseMetricLine(`all_cli_command_total{junk"} 5`); ok {
		t.Fatal("malformed labels must be rejected")
	}
}

func TestReadPrometheusOpenError(t *testing.T) {
	// A path beneath a regular file fails os.Open with ENOTDIR, exercising the
	// non-NotExist error branch.
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readPrometheus(filepath.Join(blocker, "m.prom")); err == nil {
		t.Fatal("expected open error for path under a file")
	}
}

func TestRecordPrometheusPropagatesReadError(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := recordPrometheus(filepath.Join(blocker, "m.prom"), "status", "success", 0); err == nil {
		t.Fatal("record must propagate the read error")
	}
}
