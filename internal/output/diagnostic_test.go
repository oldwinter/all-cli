package output

import (
	"strings"
	"testing"

	"github.com/oldwinter/all-cli/internal/model"
)

func TestPrintDiagnosticReportEmpty(t *testing.T) {
	var out strings.Builder
	PrintDiagnosticReport(&out, model.DiagnosticReport{
		Summary: model.DiagnosticSummary{Total: 0},
		Profile: "agent",
	})
	got := out.String()
	if !strings.Contains(got, "Diagnostics: total=0 info=0 warning=0 error=0 profile=agent") {
		t.Fatalf("missing summary line: %q", got)
	}
	if !strings.Contains(got, "No diagnostics found.") {
		t.Fatalf("missing empty notice: %q", got)
	}
}

func TestPrintDiagnosticReportItems(t *testing.T) {
	var out strings.Builder
	PrintDiagnosticReport(&out, model.DiagnosticReport{
		Summary: model.DiagnosticSummary{Total: 1, Warning: 1},
		Profile: "human",
		Diagnostics: []model.DiagnosticItem{
			{
				Severity:    model.DiagnosticWarning,
				RelatedTool: "kubectl",
				Problem:     "context missing",
				Evidence:    []string{"no kubeconfig"},
				SuggestedActions: []model.SuggestedAction{
					{
						ID:      "install_tool",
						Title:   "Install kubectl",
						Command: []string{"brew", "install", "kubectl"},
					},
					{ID: "rerun", Title: "   "},
				},
				SafeToAutofix: true,
			},
		},
	})
	got := out.String()
	for _, want := range []string{
		"[warning] kubectl: context missing",
		"  evidence: no kubeconfig",
		"  action: install_tool - Install kubectl (brew install kubectl)",
		"  action: rerun\n",
		"  safe_to_autofix: true",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q: %q", want, got)
		}
	}
}

func TestPrintDoctorReportWrapsDiagnostics(t *testing.T) {
	var out strings.Builder
	PrintDoctorReport(&out, model.DiagnosticReport{
		Summary: model.DiagnosticSummary{Total: 1, Error: 1},
		Profile: "ci",
	})
	got := out.String()
	if !strings.HasPrefix(got, "Doctor\n") {
		t.Fatalf("missing Doctor header: %q", got)
	}
	if !strings.Contains(got, "Diagnostics: total=1") {
		t.Fatalf("missing diagnostic body: %q", got)
	}
}

func TestPrintFixPlanEmptyAndItems(t *testing.T) {
	var empty strings.Builder
	PrintFixPlan(&empty, model.FixPlan{DryRun: true})
	if got := empty.String(); !strings.Contains(got, "Fix plan: dry_run=true total=0 supported=0 blocked=0") || !strings.Contains(got, "No fixes planned.") {
		t.Fatalf("empty plan output = %q", got)
	}

	var out strings.Builder
	PrintFixPlan(&out, model.FixPlan{
		DryRun:  true,
		Summary: model.FixPlanSummary{Total: 1, Supported: 1},
		Items: []model.FixPlanItem{
			{
				RelatedTool: "gh",
				Action:      model.SuggestedAction{ID: "install_tool"},
				Supported:   true,
				WillRun:     false,
				Reason:      "dry-run",
			},
		},
	})
	got := out.String()
	if !strings.Contains(got, "total=1 supported=1") {
		t.Fatalf("missing plan summary: %q", got)
	}
	if !strings.Contains(got, "- gh install_tool supported=true will_run=false reason=dry-run") {
		t.Fatalf("missing item line: %q", got)
	}
}

func TestPrintSnapshotDiffSortsChanges(t *testing.T) {
	var out strings.Builder
	PrintSnapshotDiff(&out, model.SnapshotDiffReport{
		Summary: model.SnapshotDiffSummary{Added: 1, Changed: 1},
		Changes: []model.SnapshotToolChange{
			{ToolID: "zulu", ChangeType: model.SnapshotChangeChanged, Fields: []string{"context"}},
			{ToolID: "alpha", ChangeType: model.SnapshotChangeAdded},
		},
	})
	got := out.String()
	if !strings.Contains(got, "Snapshot diff: added=1 removed=0 changed=1") {
		t.Fatalf("missing summary: %q", got)
	}
	alpha := strings.Index(got, "- alpha added")
	zulu := strings.Index(got, "- zulu changed fields=context")
	if alpha < 0 || zulu < 0 || alpha > zulu {
		t.Fatalf("changes not sorted by tool ID: %q", got)
	}
}

func TestPrintSnapshotDiffEmpty(t *testing.T) {
	var out strings.Builder
	PrintSnapshotDiff(&out, model.SnapshotDiffReport{})
	if got := out.String(); !strings.Contains(got, "No changes.") {
		t.Fatalf("empty diff output = %q", got)
	}
}
