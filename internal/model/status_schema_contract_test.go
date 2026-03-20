package model

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found from cwd")
		}
		dir = parent
	}
}

func TestStatusReportSchemaFileIsValidJSON(t *testing.T) {
	t.Parallel()

	path := filepath.Join(moduleRoot(t), "schemas", "status-report-v0.1.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var doc any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("schema file: %v", err)
	}
}

func TestStatusReportJSONKeyContract(t *testing.T) {
	t.Parallel()

	report := NewStatusReport(1)
	report.GeneratedAt = time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	report.Tools[0] = ToolSummary{
		ID:              "example",
		DisplayName:     "Example",
		Category:        "test",
		Installed:       true,
		ConfiguredState: ConfiguredNA,
		Configured:      true,
		Capabilities:    Capability{},
	}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var top map[string]json.RawMessage
	if err := json.Unmarshal(data, &top); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, key := range []string{"schema_version", "generated_at", "legend", "tools"} {
		if _, ok := top[key]; !ok {
			t.Errorf("missing top-level key %q in marshaled status JSON", key)
		}
	}

	var tools []json.RawMessage
	if err := json.Unmarshal(top["tools"], &tools); err != nil {
		t.Fatalf("tools array: %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("tools len = %d, want 1", len(tools))
	}

	var row map[string]json.RawMessage
	if err := json.Unmarshal(tools[0], &row); err != nil {
		t.Fatalf("tool row: %v", err)
	}
	for _, key := range []string{
		"id", "display_name", "category", "installed",
		"configured_state", "configured", "capabilities",
	} {
		if _, ok := row[key]; !ok {
			t.Errorf("missing tool key %q", key)
		}
	}
}
