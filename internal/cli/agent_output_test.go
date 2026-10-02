package cli

import (
	"strings"
	"testing"

	"github.com/oldwinter/all-cli/internal/model"
)

func TestValidateAgentProfile(t *testing.T) {
	t.Parallel()

	for _, ok := range []string{"", "agent", "human", "ci", " HUMAN ", "Agent"} {
		if err := validateAgentProfile(ok); err != nil {
			t.Fatalf("profile %q: unexpected error %v", ok, err)
		}
	}
	if err := validateAgentProfile("bogus"); err == nil || !strings.Contains(err.Error(), "invalid --profile") {
		t.Fatalf("expected invalid profile error, got %v", err)
	}
}

func TestBuildDiagnosticReportRejectsBadProfile(t *testing.T) {
	t.Parallel()

	opts := &rootOptions{}
	if _, err := buildDiagnosticReport(nil, opts, nil, "", "bogus"); err == nil {
		t.Fatal("expected profile validation error before any evaluation")
	}
}

func TestPrintDiagnosticReportEmpty(t *testing.T) {
	t.Parallel()

	var buf strings.Builder
	report := model.DiagnosticReport{Profile: "agent"}
	printDiagnosticReport(&buf, report)
	got := buf.String()
	for _, needle := range []string{"Diagnostics: total=0", "No diagnostics found.", "all-cli catalog", "all-cli fix --dry-run"} {
		if !strings.Contains(got, needle) {
			t.Fatalf("output missing %q:\n%s", needle, got)
		}
	}
}

func TestPrintDiagnosticReportPopulated(t *testing.T) {
	t.Parallel()

	var buf strings.Builder
	report := model.DiagnosticReport{
		Profile: "ci",
		Summary: model.DiagnosticSummary{Total: 1, Error: 1},
		Diagnostics: []model.DiagnosticItem{{
			ID:          "aws_broken",
			Severity:    model.DiagnosticError,
			RelatedTool: "aws",
			Problem:     "aws configure failed",
			Evidence:    []string{"exit=1"},
			SuggestedActions: []model.SuggestedAction{{
				ID:      "install_aws",
				Title:   "Install aws",
				Command: []string{"brew", "install", "awscli"},
				Mutates: true,
			}},
			SafeToAutofix: true,
		}},
	}
	printDiagnosticReport(&buf, report)
	got := buf.String()
	for _, needle := range []string{
		"total=1 info=0 warning=0 error=1 profile=ci",
		"[error] aws: aws configure failed",
		"evidence: exit=1",
		"action: install_aws - Install aws (brew install awscli)",
		"safe_to_autofix: true",
	} {
		if !strings.Contains(got, needle) {
			t.Fatalf("output missing %q:\n%s", needle, got)
		}
	}
}

func TestPrintDoctorReportHasHeader(t *testing.T) {
	t.Parallel()

	var buf strings.Builder
	printDoctorReport(&buf, model.DiagnosticReport{Profile: "human"})
	got := buf.String()
	if !strings.HasPrefix(got, "Doctor\n") {
		t.Fatalf("expected Doctor header, got:\n%s", got)
	}
}

func TestPrintFixPlanEmptyAndPopulated(t *testing.T) {
	t.Parallel()

	var empty strings.Builder
	printFixPlan(&empty, model.FixPlan{DryRun: true})
	if !strings.Contains(empty.String(), "dry_run=true") || !strings.Contains(empty.String(), "No fixes planned.") {
		t.Fatalf("empty plan output:\n%s", empty.String())
	}

	var buf strings.Builder
	printFixPlan(&buf, model.FixPlan{
		DryRun: true,
		Summary: model.FixPlanSummary{
			Total:     2,
			Supported: 1,
			Blocked:   1,
		},
		Items: []model.FixPlanItem{
			{
				RelatedTool: "aws",
				Action:      model.SuggestedAction{ID: "install_aws"},
				Supported:   true,
				WillRun:     true,
				Reason:      "installer available",
			},
			{
				RelatedTool: "kubectl",
				Action:      model.SuggestedAction{ID: "configure_context"},
				Supported:   false,
				WillRun:     false,
				Reason:      "no supported fix",
			},
		},
	})
	got := buf.String()
	for _, needle := range []string{
		"total=2 supported=1 blocked=1",
		"- aws install_aws supported=true will_run=true reason=installer available",
		"- kubectl configure_context supported=false will_run=false reason=no supported fix",
	} {
		if !strings.Contains(got, needle) {
			t.Fatalf("output missing %q:\n%s", needle, got)
		}
	}
}
