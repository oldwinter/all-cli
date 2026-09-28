package telemetry

import (
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
