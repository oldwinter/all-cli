package model

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
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

const statusReportSchemaURL = "https://github.com/oldwinter/all-cli/schemas/status-report-v0.1.json"

func TestStatusReportMarshalsAgainstStatusSchema(t *testing.T) {
	t.Parallel()

	path := filepath.Join(moduleRoot(t), "schemas", "status-report-v0.1.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var schemaDoc any
	if err := json.Unmarshal(raw, &schemaDoc); err != nil {
		t.Fatalf("parse schema: %v", err)
	}

	c := jsonschema.NewCompiler()
	if err := c.AddResource(statusReportSchemaURL, schemaDoc); err != nil {
		t.Fatalf("add schema resource: %v", err)
	}
	sch, err := c.Compile(statusReportSchemaURL)
	if err != nil {
		t.Fatalf("compile schema: %v", err)
	}

	report := NewStatusReport(1)
	report.GeneratedAt = time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	report.Tools[0] = ToolSummary{
		ID:              "example",
		DisplayName:     "Example",
		Category:        "test",
		Installed:       true,
		InstallPath:     "/usr/bin/example",
		ConfiguredState: ConfiguredYes,
		Configured:      true,
		Capabilities: Capability{
			HasContexts: true,
			CanSwitch:   false,
		},
		Current: map[string]string{"context": "dev"},
		Metadata: ToolMetadata{
			Purpose:        "Exercise every metadata field for schema validation.",
			ConfiguredWhen: "A synthetic positive case for contract testing.",
			CurrentFieldDescriptions: map[string]string{
				"context": "Synthetic current field for schema validation.",
			},
			AgentActions: []string{"inspect_status"},
			Notes:        []string{"Synthetic note for schema validation."},
		},
		Warnings: []string{"synthetic warning"},
		Errors:   []string{"synthetic error"},
	}

	payload, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	var instance any
	if err := json.Unmarshal(payload, &instance); err != nil {
		t.Fatalf("unmarshal instance: %v", err)
	}

	if err := sch.Validate(instance); err != nil {
		t.Fatalf("instance does not validate against %s: %v", statusReportSchemaURL, err)
	}
}
