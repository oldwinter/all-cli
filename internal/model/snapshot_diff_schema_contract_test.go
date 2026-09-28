package model

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

const snapshotDiffReportSchemaURL = "https://github.com/oldwinter/all-cli/schemas/snapshot-diff-report-v0.1.json"

func compileSnapshotDiffSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()

	c := jsonschema.NewCompiler()
	resources := map[string]string{
		statusReportSchemaURL:       "status-report-v0.1.json",
		snapshotDiffReportSchemaURL: "snapshot-diff-report-v0.1.json",
	}
	for url, file := range resources {
		raw, err := os.ReadFile(filepath.Join(moduleRoot(t), "schemas", file))
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		var doc any
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		if err := c.AddResource(url, doc); err != nil {
			t.Fatalf("add %s: %v", file, err)
		}
	}
	sch, err := c.Compile(snapshotDiffReportSchemaURL)
	if err != nil {
		t.Fatalf("compile schema: %v", err)
	}
	return sch
}

func snapshotDiffInstance(t *testing.T, report SnapshotDiffReport) any {
	t.Helper()

	payload, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	var instance any
	if err := json.Unmarshal(payload, &instance); err != nil {
		t.Fatalf("unmarshal instance: %v", err)
	}
	return instance
}

func TestSnapshotDiffReportMarshalsAgainstSchema(t *testing.T) {
	t.Parallel()

	sch := compileSnapshotDiffSchema(t)

	before := ToolSummary{
		ID:              "kubectl",
		DisplayName:     "kubectl",
		Category:        "k8s",
		Installed:       true,
		ConfiguredState: ConfiguredYes,
		Configured:      true,
		Capabilities:    Capability{HasContexts: true, CanSwitch: true},
		Current:         map[string]string{"context": "dev"},
	}
	after := before
	after.ConfiguredState = ConfiguredNo
	after.Configured = false
	after.Current = map[string]string{"context": "prod"}
	added := ToolSummary{
		ID:              "gh",
		DisplayName:     "gh",
		Category:        "code",
		Installed:       true,
		ConfiguredState: ConfiguredYes,
		Configured:      true,
		Capabilities:    Capability{HasContexts: true, CanSwitch: true},
	}

	report := SnapshotDiffReport{
		SchemaVersion: SnapshotDiffSchemaVersionV01,
		GeneratedAt:   time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC),
		Summary:       SnapshotDiffSummary{Added: 1, Removed: 0, Changed: 1},
		Changes: []SnapshotToolChange{
			{
				ToolID:     "gh",
				ChangeType: SnapshotChangeAdded,
				After:      &added,
			},
			{
				ToolID:     "kubectl",
				ChangeType: SnapshotChangeChanged,
				Fields:     []string{"configured_state", "configured", "current"},
				Before:     &before,
				After:      &after,
			},
		},
	}

	if err := sch.Validate(snapshotDiffInstance(t, report)); err != nil {
		t.Fatalf("instance does not validate against %s: %v", snapshotDiffReportSchemaURL, err)
	}
}

func TestSnapshotDiffReportEmptyChangesValid(t *testing.T) {
	t.Parallel()

	sch := compileSnapshotDiffSchema(t)
	report := SnapshotDiffReport{
		SchemaVersion: SnapshotDiffSchemaVersionV01,
		GeneratedAt:   time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC),
		Summary:       SnapshotDiffSummary{},
		Changes:       []SnapshotToolChange{},
	}
	if err := sch.Validate(snapshotDiffInstance(t, report)); err != nil {
		t.Fatalf("empty diff should validate: %v", err)
	}
}

func TestSnapshotDiffSchemaRejectsWrongVersion(t *testing.T) {
	t.Parallel()

	sch := compileSnapshotDiffSchema(t)
	report := SnapshotDiffReport{
		SchemaVersion: "v0.1",
		GeneratedAt:   time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC),
		Changes:       []SnapshotToolChange{},
	}
	if err := sch.Validate(snapshotDiffInstance(t, report)); err == nil {
		t.Fatal("schema accepted wrong schema_version")
	}
}
