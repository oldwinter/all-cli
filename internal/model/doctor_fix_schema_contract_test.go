package model

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

const doctorFixReportSchemaURL = "https://github.com/oldwinter/all-cli/schemas/doctor-fix-report-v0.1.json"

func compileDoctorFixSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()

	c := jsonschema.NewCompiler()
	resources := map[string]string{
		statusReportSchemaURL:     "status-report-v0.1.json",
		diagnosticReportSchemaURL: "diagnostic-report-v0.1.json",
		doctorFixReportSchemaURL:  "doctor-fix-report-v0.1.json",
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
	sch, err := c.Compile(doctorFixReportSchemaURL)
	if err != nil {
		t.Fatalf("compile schema: %v", err)
	}
	return sch
}

func doctorFixInstance(t *testing.T, report DoctorFixReport) any {
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

func TestDoctorFixReportMarshalsAgainstDoctorFixSchema(t *testing.T) {
	t.Parallel()

	sch := compileDoctorFixSchema(t)
	report := DoctorFixReport{
		SchemaVersion: DoctorFixSchemaVersionV01,
		Report: DiagnosticReport{
			SchemaVersion:       DiagnosticSchemaVersionV01,
			GeneratedAt:         time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC),
			SourceSchemaVersion: SchemaVersionV01,
			Profile:             "human",
			Summary:             DiagnosticSummary{Total: 1, Info: 1},
			Diagnostics: []DiagnosticItem{{
				ID:          "gh.not_installed",
				Severity:    DiagnosticInfo,
				Problem:     "GitHub CLI is not installed.",
				RelatedTool: "gh",
				SuggestedActions: []SuggestedAction{{
					ID:      "install_tool",
					Title:   "Install GitHub CLI",
					Mutates: true,
				}},
			}},
		},
		Fixes: DoctorFixRun{
			Installer: "auto",
			Summary:   DoctorFixSummary{Total: 2, Supported: 2, Installed: 1, Failed: 1},
			Items: []DoctorFixItem{
				{ToolID: "gh", Installer: "brew", Command: []string{"brew", "install", "gh"}, Supported: true, Status: DoctorFixInstalled},
				{ToolID: "codex", Installer: "npm", Command: []string{"npm", "install", "-g", "@openai/codex"}, Supported: true, Status: DoctorFixFailed, Reason: "exit status 1", ExitCode: 1},
			},
		},
	}

	if err := sch.Validate(doctorFixInstance(t, report)); err != nil {
		t.Fatalf("instance does not validate against %s: %v", doctorFixReportSchemaURL, err)
	}
}

func TestDoctorFixSchemaRejectsUnknownStatus(t *testing.T) {
	t.Parallel()

	sch := compileDoctorFixSchema(t)
	report := DoctorFixReport{
		SchemaVersion: DoctorFixSchemaVersionV01,
		Report: DiagnosticReport{
			SchemaVersion:       DiagnosticSchemaVersionV01,
			SourceSchemaVersion: SchemaVersionV01,
			Diagnostics:         []DiagnosticItem{},
		},
		Fixes: DoctorFixRun{
			Installer: "auto",
			Items:     []DoctorFixItem{{ToolID: "gh", Status: "pending"}},
		},
	}

	err := sch.Validate(doctorFixInstance(t, report))
	if err == nil || !strings.Contains(err.Error(), "status") {
		t.Fatalf("Validate() error = %v, want status enum violation", err)
	}
}
