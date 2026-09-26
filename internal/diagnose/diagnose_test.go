package diagnose

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/oldwinter/all-cli/internal/model"
)

func TestDiffSnapshotsCollectionFields(t *testing.T) {
	for _, tt := range []struct {
		name   string
		before string
		after  string
		fields []string
	}{
		{"both absent", ``, ``, nil},
		{"empty current", ``, `,"current":{}`, nil},
		{"empty warnings", ``, `,"warnings":[]`, nil},
		{"empty errors", ``, `,"errors":[]`, nil},
		{"all empty", ``, `,"current":{},"warnings":[],"errors":[]`, nil},
		{"both empty", `,"current":{},"warnings":[],"errors":[]`, `,"current":{},"warnings":[],"errors":[]`, nil},
		{"current from absent", ``, `,"current":{"context":"dev"}`, []string{"current"}},
		{"current from empty", `,"current":{}`, `,"current":{"context":"dev"}`, []string{"current"}},
		{"warnings from absent", ``, `,"warnings":["warning"]`, []string{"warnings"}},
		{"warnings from empty", `,"warnings":[]`, `,"warnings":["warning"]`, []string{"warnings"}},
		{"errors from absent", ``, `,"errors":["error"]`, []string{"errors"}},
		{"errors from empty", `,"errors":[]`, `,"errors":["error"]`, []string{"errors"}},
		{"current value changed", `,"current":{"context":"dev"}`, `,"current":{"context":"prod"}`, []string{"current"}},
		{"current key changed", `,"current":{"context":""}`, `,"current":{"namespace":""}`, []string{"current"}},
		{"warning changed", `,"warnings":["first"]`, `,"warnings":["second"]`, []string{"warnings"}},
		{"error changed", `,"errors":["first"]`, `,"errors":["second"]`, []string{"errors"}},
		{"warning order changed", `,"warnings":["first","second"]`, `,"warnings":["second","first"]`, []string{"warnings"}},
		{"error order changed", `,"errors":["first","second"]`, `,"errors":["second","first"]`, []string{"errors"}},
		{"populated unchanged", `,"current":{"context":"dev"},"warnings":["warning"],"errors":["error"]`, `,"current":{"context":"dev"},"warnings":["warning"],"errors":["error"]`, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var before, after model.StatusReport
			if err := json.Unmarshal([]byte(`{"tools":[{"id":"kubectl"`+tt.before+`}]}`), &before); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(`{"tools":[{"id":"kubectl"`+tt.after+`}]}`), &after); err != nil {
				t.Fatal(err)
			}
			for _, direction := range []struct {
				name          string
				before, after model.StatusReport
			}{
				{"forward", before, after},
				{"reverse", after, before},
			} {
				t.Run(direction.name, func(t *testing.T) {
					report := DiffSnapshots(direction.before, direction.after)
					wantChanges := []model.SnapshotToolChange{}
					if len(tt.fields) > 0 {
						wantChanges = append(wantChanges, model.SnapshotToolChange{
							ToolID: "kubectl", ChangeType: model.SnapshotChangeChanged, Fields: tt.fields,
							Before: &direction.before.Tools[0], After: &direction.after.Tools[0],
						})
					}
					if !reflect.DeepEqual(report.Changes, wantChanges) {
						t.Errorf("changes = %#v, want %#v", report.Changes, wantChanges)
					}
					if want := (model.SnapshotDiffSummary{Changed: len(wantChanges)}); report.Summary != want {
						t.Errorf("summary = %#v, want %#v", report.Summary, want)
					}
				})
			}
		})
	}
}

func TestDiffSnapshotsChangesSortedByToolID(t *testing.T) {
	added := model.ToolSummary{ID: "aws"}
	removed := model.ToolSummary{ID: "rg"}
	unchanged := model.ToolSummary{ID: "docker"}
	oldTool := model.ToolSummary{ID: "kubectl"}
	newTool := model.ToolSummary{
		ID: "kubectl", DisplayName: "Kubernetes", Category: "containers",
		Installed: true, InstallPath: "/usr/local/bin/kubectl",
		ConfiguredState: model.ConfiguredYes, Configured: true,
		Capabilities: model.Capability{HasContexts: true, CanSwitch: true},
		Current:      map[string]string{"context": "dev"},
		Warnings:     []string{"warning"}, Errors: []string{"error"},
	}
	before := model.StatusReport{Tools: []model.ToolSummary{removed, unchanged, oldTool}}
	after := model.StatusReport{Tools: []model.ToolSummary{newTool, added, unchanged}}

	report := DiffSnapshots(before, after)

	if report.SchemaVersion != model.SnapshotDiffSchemaVersionV01 || report.GeneratedAt.IsZero() {
		t.Errorf("unexpected report metadata: %#v", report)
	}
	if want := (model.SnapshotDiffSummary{Added: 1, Removed: 1, Changed: 1}); report.Summary != want {
		t.Errorf("summary = %#v, want %#v", report.Summary, want)
	}
	wantChanges := []model.SnapshotToolChange{
		{ToolID: "aws", ChangeType: model.SnapshotChangeAdded, After: &added},
		{
			ToolID: "kubectl", ChangeType: model.SnapshotChangeChanged, Before: &oldTool, After: &newTool,
			Fields: []string{"display_name", "category", "installed", "install_path", "configured_state", "configured", "capabilities", "current", "warnings", "errors"},
		},
		{ToolID: "rg", ChangeType: model.SnapshotChangeRemoved, Before: &removed},
	}
	if !reflect.DeepEqual(report.Changes, wantChanges) {
		t.Fatalf("changes = %#v, want %#v", report.Changes, wantChanges)
	}
}

func TestIndexToolsSkipsBlankIDs(t *testing.T) {
	tool := model.ToolSummary{ID: " kubectl ", Installed: true}
	got := indexTools([]model.ToolSummary{{ID: ""}, {ID: " \t\n"}, tool})
	want := map[string]model.ToolSummary{"kubectl": tool}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("indexTools = %#v, want %#v", got, want)
	}
}

func TestGenerateCreatesInfoDiagnosticForMissingTool(t *testing.T) {
	status := model.NewStatusReport(1)
	status.GeneratedAt = time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	status.Tools[0] = model.ToolSummary{
		ID:              "rg",
		DisplayName:     "ripgrep",
		Category:        "navigation",
		Installed:       false,
		ConfiguredState: model.ConfiguredUnknown,
		Metadata: model.ToolMetadata{
			AgentActions: []string{"inspect_status"},
		},
	}

	report := Generate(status, Options{Profile: ProfileAgent})

	if report.SchemaVersion != model.DiagnosticSchemaVersionV01 {
		t.Fatalf("schema version = %q, want %q", report.SchemaVersion, model.DiagnosticSchemaVersionV01)
	}
	if report.Profile != ProfileAgent {
		t.Fatalf("profile = %q, want %q", report.Profile, ProfileAgent)
	}
	if report.Summary.Total != 1 || report.Summary.Info != 1 {
		t.Fatalf("unexpected summary: %#v", report.Summary)
	}
	if len(report.Diagnostics) != 1 {
		t.Fatalf("diagnostics len = %d, want 1", len(report.Diagnostics))
	}
	item := report.Diagnostics[0]
	if item.RelatedTool != "rg" || item.Severity != model.DiagnosticInfo {
		t.Fatalf("unexpected diagnostic item: %#v", item)
	}
	if !strings.Contains(item.Problem, "not installed") {
		t.Fatalf("problem should explain missing binary, got %q", item.Problem)
	}
	if item.SafeToAutofix {
		t.Fatalf("missing tools should not be marked safe to autofix")
	}
	if len(item.SuggestedActions) == 0 || item.SuggestedActions[0].ID != "install_tool" {
		t.Fatalf("expected install action, got %#v", item.SuggestedActions)
	}
}

func TestGenerateCreatesWarningForUnconfiguredInstalledTool(t *testing.T) {
	status := model.NewStatusReport(1)
	status.Tools[0] = model.ToolSummary{
		ID:              "docker",
		DisplayName:     "Docker",
		Category:        "containers",
		Installed:       true,
		InstallPath:     "/usr/local/bin/docker",
		ConfiguredState: model.ConfiguredNo,
		Metadata: model.ToolMetadata{
			ConfiguredWhen: "At least one Docker context is available.",
			AgentActions:   []string{"inspect_status", "show_current"},
		},
	}

	report := Generate(status, Options{Profile: ProfileCI})

	if report.Profile != ProfileCI {
		t.Fatalf("profile = %q, want %q", report.Profile, ProfileCI)
	}
	if report.Summary.Warning != 1 {
		t.Fatalf("expected one warning, got %#v", report.Summary)
	}
	item := report.Diagnostics[0]
	if item.Severity != model.DiagnosticWarning {
		t.Fatalf("severity = %q, want %q", item.Severity, model.DiagnosticWarning)
	}
	if !strings.Contains(strings.Join(item.Evidence, "\n"), "configured_state=no") {
		t.Fatalf("expected configured_state evidence, got %#v", item.Evidence)
	}
	if len(item.SuggestedActions) == 0 || item.SuggestedActions[0].ID != "configure_tool" {
		t.Fatalf("expected configure action, got %#v", item.SuggestedActions)
	}
}

func TestGenerateCreatesErrorDiagnosticForCollectionErrors(t *testing.T) {
	status := model.NewStatusReport(1)
	status.Tools[0] = model.ToolSummary{
		ID:              "aws",
		DisplayName:     "AWS CLI",
		Category:        "cloud",
		Installed:       true,
		ConfiguredState: model.ConfiguredUnknown,
		Errors:          []string{"context deadline exceeded"},
	}

	report := Generate(status, Options{})

	if report.Summary.Error != 1 {
		t.Fatalf("expected one error, got %#v", report.Summary)
	}
	item := report.Diagnostics[0]
	if item.Severity != model.DiagnosticError {
		t.Fatalf("severity = %q, want %q", item.Severity, model.DiagnosticError)
	}
	if !strings.Contains(strings.Join(item.Evidence, "\n"), "context deadline exceeded") {
		t.Fatalf("expected error evidence, got %#v", item.Evidence)
	}
	if len(item.SuggestedActions) == 0 || item.SuggestedActions[0].ID != "rerun_with_timeout" {
		t.Fatalf("expected timeout/retry action, got %#v", item.SuggestedActions)
	}
}

func TestGenerateSuggestsUpgradeForOldGHJSONFlag(t *testing.T) {
	status := model.NewStatusReport(1)
	status.Tools[0] = model.ToolSummary{
		ID:              "gh",
		DisplayName:     "gh",
		Installed:       true,
		ConfiguredState: model.ConfiguredUnknown,
		Errors:          []string{"unknown flag: --json\n\nUsage:  gh auth status [flags]..."},
	}

	report := Generate(status, Options{})
	if report.Summary.Error != 1 || len(report.Diagnostics) == 0 {
		t.Fatalf("expected one collection error, got %#v", report)
	}
	item := report.Diagnostics[0]
	if len(item.SuggestedActions) == 0 {
		t.Fatalf("expected a suggested action, got %#v", item)
	}
	action := item.SuggestedActions[0]
	if action.ID == "rerun_with_timeout" || strings.Contains(action.Title, "timeout") {
		t.Fatalf("old gh JSON flag should not be diagnosed as timeout: %#v", action)
	}
	if action.ID != "upgrade_or_check_gh_auth" {
		t.Fatalf("expected upgrade/check-auth action, got %#v", action)
	}
	joined := action.Title + " " + action.Description + " " + strings.Join(action.Command, " ")
	if !strings.Contains(joined, "gh auth status") && !strings.Contains(strings.ToLower(joined), "upgrade") {
		t.Fatalf("expected upgrade or gh auth status guidance, got %#v", action)
	}
}

func TestBuildFixPlanIsDryRunOnlyAndNonMutating(t *testing.T) {
	status := model.NewStatusReport(1)
	status.Tools[0] = model.ToolSummary{
		ID:              "gh",
		DisplayName:     "GitHub CLI",
		Installed:       true,
		ConfiguredState: model.ConfiguredNo,
	}
	report := Generate(status, Options{})

	plan := BuildFixPlan(report, FixOptions{DryRun: true})

	if !plan.DryRun {
		t.Fatalf("fix plan should preserve dry_run=true")
	}
	if plan.Summary.Total != 1 || plan.Summary.Supported != 0 || plan.Summary.Blocked != 1 {
		t.Fatalf("unexpected fix summary: %#v", plan.Summary)
	}
	if len(plan.Items) != 1 {
		t.Fatalf("items len = %d, want 1", len(plan.Items))
	}
	item := plan.Items[0]
	if item.WillRun || item.Mutates {
		t.Fatalf("dry-run baseline must not run or mutate, got %#v", item)
	}
	if !strings.Contains(item.Reason, "not allowlisted") {
		t.Fatalf("expected allowlist reason, got %q", item.Reason)
	}
}
