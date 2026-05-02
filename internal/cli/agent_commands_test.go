package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
	"github.com/oldwinter/all-cli/internal/tools"
)

func stubAgentStatusEvaluation(t *testing.T) {
	t.Helper()

	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "aws", DisplayName: "AWS CLI", Category: "cloud", Binary: "aws"},
		{ID: "kubectl", DisplayName: "kubectl", Category: "k8s", Binary: "kubectl"},
	})

	oldEvaluate := evaluateToolSummary
	evaluateToolSummary = func(_ context.Context, def tools.ToolDefinition, _ execx.Runner) model.ToolSummary {
		switch def.ID {
		case "aws":
			return model.ToolSummary{
				ID:              "aws",
				DisplayName:     "AWS CLI",
				Category:        "cloud",
				Installed:       true,
				ConfiguredState: model.ConfiguredUnknown,
				Errors:          []string{"context deadline exceeded"},
			}
		case "kubectl":
			return model.ToolSummary{
				ID:              "kubectl",
				DisplayName:     "kubectl",
				Category:        "k8s",
				Installed:       true,
				ConfiguredState: model.ConfiguredYes,
				Configured:      true,
				Current:         map[string]string{"context": "orbstack"},
			}
		default:
			t.Fatalf("unexpected tool %s", def.ID)
			return model.ToolSummary{}
		}
	}
	t.Cleanup(func() {
		evaluateToolSummary = oldEvaluate
	})
	stubShowStatusSpinner(t, false)
}

func TestDiagnoseCommandJSONUsesToolsFilter(t *testing.T) {
	stubAgentStatusEvaluation(t)

	opts := &rootOptions{JSON: true, Timeout: time.Second}
	stdout, stderr, err := executeTestCommand(t, newDiagnoseCommand(opts, cliFakeRunner{}), "--tools", "aws", "--profile", "agent")
	if err != nil {
		t.Fatalf("diagnose: %v", err)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	var got model.DiagnosticReport
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode diagnose json: %v", err)
	}
	if got.SchemaVersion != model.DiagnosticSchemaVersionV01 || got.Profile != "agent" {
		t.Fatalf("unexpected report header: %#v", got)
	}
	if len(got.Tools) != 1 || got.Tools[0].ID != "aws" {
		t.Fatalf("expected only aws tool, got %#v", got.Tools)
	}
	if got.Summary.Error != 1 || len(got.Diagnostics) != 1 {
		t.Fatalf("expected one error diagnostic, got %#v", got)
	}
}

func TestDiagnoseCommandPlainPrintsDiagnosticEvidence(t *testing.T) {
	stubAgentStatusEvaluation(t)

	opts := &rootOptions{Timeout: time.Second}
	stdout, _, err := executeTestCommand(t, newDiagnoseCommand(opts, cliFakeRunner{}), "--tools", "aws")
	if err != nil {
		t.Fatalf("diagnose: %v", err)
	}
	for _, needle := range []string{"Diagnostics", "aws", "context deadline exceeded", "rerun_with_timeout"} {
		if !strings.Contains(stdout, needle) {
			t.Fatalf("stdout missing %q:\n%s", needle, stdout)
		}
	}
}

func TestDoctorCommandJSONUsesDiagnosticReport(t *testing.T) {
	stubAgentStatusEvaluation(t)

	opts := &rootOptions{JSON: true, Timeout: time.Second}
	stdout, _, err := executeTestCommand(t, newDoctorCommand(opts, cliFakeRunner{}), "--tools", "aws")
	if err != nil {
		t.Fatalf("doctor: %v", err)
	}
	var got model.DiagnosticReport
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode doctor json: %v", err)
	}
	if got.Summary.Error != 1 {
		t.Fatalf("expected doctor to return diagnostic report, got %#v", got)
	}
}

func TestFixCommandRequiresDryRun(t *testing.T) {
	stubAgentStatusEvaluation(t)

	opts := &rootOptions{Timeout: time.Second}
	_, _, err := executeTestCommand(t, newFixCommand(opts, cliFakeRunner{}), "--tools", "aws")
	if err == nil || !strings.Contains(err.Error(), "--dry-run") {
		t.Fatalf("expected --dry-run error, got %v", err)
	}
}

func TestFixCommandDryRunJSONReturnsBlockedPlan(t *testing.T) {
	stubAgentStatusEvaluation(t)

	opts := &rootOptions{JSON: true, Timeout: time.Second}
	stdout, _, err := executeTestCommand(t, newFixCommand(opts, cliFakeRunner{}), "--dry-run", "--tools", "aws")
	if err != nil {
		t.Fatalf("fix --dry-run: %v", err)
	}
	var got model.FixPlan
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode fix plan: %v", err)
	}
	if !got.DryRun || got.Summary.Blocked != 1 || got.Summary.Supported != 0 {
		t.Fatalf("unexpected fix plan: %#v", got)
	}
	if len(got.Items) != 1 || got.Items[0].WillRun || got.Items[0].Mutates {
		t.Fatalf("expected non-mutating blocked item, got %#v", got.Items)
	}
}

func TestSnapshotCommandJSONReturnsStatusReport(t *testing.T) {
	stubAgentStatusEvaluation(t)

	opts := &rootOptions{JSON: true, Timeout: time.Second}
	stdout, _, err := executeTestCommand(t, newSnapshotCommand(opts, cliFakeRunner{}), "--tools", "kubectl")
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	var got model.StatusReport
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	if got.SchemaVersion != model.SchemaVersionV01 || len(got.Tools) != 1 || got.Tools[0].ID != "kubectl" {
		t.Fatalf("unexpected snapshot: %#v", got)
	}
}

func TestDiffCommandJSONReportsChangedTools(t *testing.T) {
	dir := t.TempDir()
	before := model.NewStatusReport(1)
	before.Tools[0] = model.ToolSummary{ID: "aws", DisplayName: "AWS CLI", Category: "cloud", Installed: false, ConfiguredState: model.ConfiguredUnknown}
	after := model.NewStatusReport(1)
	after.Tools[0] = model.ToolSummary{ID: "aws", DisplayName: "AWS CLI", Category: "cloud", Installed: true, ConfiguredState: model.ConfiguredYes}

	beforePath := writeStatusReportFixture(t, dir, "before.json", before)
	afterPath := writeStatusReportFixture(t, dir, "after.json", after)

	opts := &rootOptions{JSON: true, Timeout: time.Second}
	stdout, _, err := executeTestCommand(t, newDiffCommand(opts), beforePath, afterPath)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}

	var got model.SnapshotDiffReport
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode diff: %v", err)
	}
	if got.SchemaVersion != model.SnapshotDiffSchemaVersionV01 || got.Summary.Changed != 1 {
		t.Fatalf("unexpected diff summary: %#v", got)
	}
	if len(got.Changes) != 1 || got.Changes[0].ToolID != "aws" || got.Changes[0].ChangeType != model.SnapshotChangeChanged {
		t.Fatalf("unexpected diff changes: %#v", got.Changes)
	}
}

func writeStatusReportFixture(t *testing.T, dir, name string, report model.StatusReport) string {
	t.Helper()
	path := filepath.Join(dir, name)
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}
