package model

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

const fixPlanReportSchemaURL = "https://github.com/oldwinter/all-cli/schemas/fix-plan-report-v0.1.json"

func compileFixPlanSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()

	c := jsonschema.NewCompiler()
	resources := map[string]string{
		statusReportSchemaURL:  "status-report-v0.1.json",
		fixPlanReportSchemaURL: "fix-plan-report-v0.1.json",
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
	sch, err := c.Compile(fixPlanReportSchemaURL)
	if err != nil {
		t.Fatalf("compile schema: %v", err)
	}
	return sch
}

func fixPlanInstance(t *testing.T, plan FixPlan) any {
	t.Helper()

	payload, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("marshal plan: %v", err)
	}
	var instance any
	if err := json.Unmarshal(payload, &instance); err != nil {
		t.Fatalf("unmarshal instance: %v", err)
	}
	return instance
}

func TestFixPlanMarshalsAgainstSchema(t *testing.T) {
	t.Parallel()

	sch := compileFixPlanSchema(t)
	plan := FixPlan{
		SchemaVersion: FixPlanSchemaVersionV01,
		GeneratedAt:   time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC),
		DryRun:        true,
		Summary:       FixPlanSummary{Total: 2, Supported: 1, Blocked: 1},
		Items: []FixPlanItem{
			{
				DiagnosticID: "fd.missing",
				RelatedTool:  "fd",
				Action: SuggestedAction{
					ID:      "install_tool",
					Title:   "Install fd",
					Kind:    "install",
					Command: []string{"brew", "install", "fd"},
					Mutates: true,
				},
				Supported: true,
				WillRun:   true,
				Mutates:   true,
			},
			{
				DiagnosticID: "kubectl.configured_state",
				RelatedTool:  "kubectl",
				Action: SuggestedAction{
					ID:      "configure_tool",
					Title:   "Configure kubectl",
					Mutates: true,
				},
				Supported: false,
				WillRun:   false,
				Mutates:   true,
				Reason:    "no supported automatic installer for this diagnostic",
			},
		},
	}
	if err := sch.Validate(fixPlanInstance(t, plan)); err != nil {
		t.Fatalf("instance does not validate against %s: %v", fixPlanReportSchemaURL, err)
	}
}

func TestFixPlanEmptyItemsValid(t *testing.T) {
	t.Parallel()

	sch := compileFixPlanSchema(t)
	plan := FixPlan{
		SchemaVersion: FixPlanSchemaVersionV01,
		GeneratedAt:   time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC),
		DryRun:        true,
		Summary:       FixPlanSummary{},
		Items:         []FixPlanItem{},
	}
	if err := sch.Validate(fixPlanInstance(t, plan)); err != nil {
		t.Fatalf("empty plan should validate: %v", err)
	}
}

func TestFixPlanSchemaRejectsWrongVersion(t *testing.T) {
	t.Parallel()

	sch := compileFixPlanSchema(t)
	plan := FixPlan{
		SchemaVersion: "v0.1",
		GeneratedAt:   time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC),
		DryRun:        true,
		Items:         []FixPlanItem{},
	}
	if err := sch.Validate(fixPlanInstance(t, plan)); err == nil {
		t.Fatal("schema accepted wrong schema_version")
	}
}
