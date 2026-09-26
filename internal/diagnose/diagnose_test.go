package diagnose

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/oldwinter/all-cli/internal/model"
)

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

func TestDiffSnapshotsReportsAddedRemovedAndChangedToolsSortedByID(t *testing.T) {
	before := model.NewStatusReport(0)
	before.Tools = []model.ToolSummary{
		{ID: "gh", Installed: true},
		{ID: "docker", Installed: true, InstallPath: "/usr/local/bin/docker"},
		{ID: "aws", Installed: true, Current: map[string]string{"profile": "before"}},
	}
	after := model.NewStatusReport(0)
	after.Tools = []model.ToolSummary{
		{ID: "kubectl", Installed: true},
		{ID: "aws", Installed: true, Current: map[string]string{"profile": "after"}},
		{ID: "gh", Installed: true},
	}

	report := DiffSnapshots(before, after)

	if report.SchemaVersion != model.SnapshotDiffSchemaVersionV01 {
		t.Fatalf("schema version = %q, want %q", report.SchemaVersion, model.SnapshotDiffSchemaVersionV01)
	}
	if report.GeneratedAt.IsZero() {
		t.Fatalf("generated_at should be set")
	}
	wantSummary := model.SnapshotDiffSummary{Added: 1, Removed: 1, Changed: 1}
	if report.Summary != wantSummary {
		t.Fatalf("summary = %#v, want %#v", report.Summary, wantSummary)
	}
	if len(report.Changes) != 3 {
		t.Fatalf("changes len = %d, want 3: %#v", len(report.Changes), report.Changes)
	}

	tests := []struct {
		name      string
		toolID    string
		change    model.SnapshotChangeType
		fields    []string
		hasBefore bool
		hasAfter  bool
	}{
		{name: "aws changed", toolID: "aws", change: model.SnapshotChangeChanged, fields: []string{"current"}, hasBefore: true, hasAfter: true},
		{name: "docker removed", toolID: "docker", change: model.SnapshotChangeRemoved, hasBefore: true},
		{name: "kubectl added", toolID: "kubectl", change: model.SnapshotChangeAdded, hasAfter: true},
	}
	for i, tt := range tests {
		change := report.Changes[i]
		if change.ToolID != tt.toolID || change.ChangeType != tt.change {
			t.Fatalf("%s: changes[%d] = %#v", tt.name, i, change)
		}
		if !reflect.DeepEqual(change.Fields, tt.fields) {
			t.Fatalf("%s: changes[%d].Fields = %v, want %v", tt.name, i, change.Fields, tt.fields)
		}
		if (change.Before != nil) != tt.hasBefore || (change.After != nil) != tt.hasAfter {
			t.Fatalf("%s: changes[%d] before/after presence wrong: %#v", tt.name, i, change)
		}
	}

	aws := report.Changes[0]
	if aws.Before.Current["profile"] != "before" || aws.After.Current["profile"] != "after" {
		t.Fatalf("changed entry should carry each snapshot's payload, got %#v", aws)
	}
	if report.Changes[1].Before.InstallPath != "/usr/local/bin/docker" {
		t.Fatalf("removed entry should keep the previous snapshot, got %#v", report.Changes[1].Before)
	}
}

func TestDiffSnapshotsSkipsBlankToolIDs(t *testing.T) {
	before := model.NewStatusReport(0)
	before.Tools = []model.ToolSummary{
		{ID: "", Installed: true},
		{ID: "   ", Installed: false, ConfiguredState: model.ConfiguredNo},
		{ID: "gh", Installed: true},
	}
	after := model.NewStatusReport(0)
	after.Tools = []model.ToolSummary{
		{ID: "gh", Installed: true},
		{ID: " \t\n ", Installed: true},
	}

	report := DiffSnapshots(before, after)

	if report.Changes == nil {
		t.Fatalf("changes should be a non-nil empty slice, got nil")
	}
	if len(report.Changes) != 0 || report.Summary != (model.SnapshotDiffSummary{}) {
		t.Fatalf("blank tool IDs must not produce changes, got %#v", report)
	}
}

func TestDiffSnapshotsMatchesToolsByTrimmedID(t *testing.T) {
	before := model.NewStatusReport(0)
	before.Tools = []model.ToolSummary{{ID: "  gh  ", Installed: true}}
	after := model.NewStatusReport(0)
	after.Tools = []model.ToolSummary{{ID: "gh", Installed: true}}

	report := DiffSnapshots(before, after)
	if len(report.Changes) != 0 {
		t.Fatalf("trimmed IDs should match, got %#v", report.Changes)
	}
}

func TestDiffSnapshotsIndexesDuplicateIDsLastWins(t *testing.T) {
	tests := []struct {
		name       string
		before     []model.ToolSummary
		after      []model.ToolSummary
		wantFields []string
	}{
		{
			name: "last before duplicate wins over earlier entry",
			before: []model.ToolSummary{
				{ID: "gh", Installed: false},
				{ID: "gh", Installed: true, InstallPath: "/bin/gh"},
			},
			after: []model.ToolSummary{
				{ID: "gh", Installed: true, InstallPath: "/bin/gh"},
			},
		},
		{
			name: "last before duplicate decides the diff",
			before: []model.ToolSummary{
				{ID: "aws", ConfiguredState: model.ConfiguredNo},
				{ID: "aws", ConfiguredState: model.ConfiguredYes, Configured: true},
			},
			after: []model.ToolSummary{
				{ID: "aws", ConfiguredState: model.ConfiguredNo},
			},
			wantFields: []string{"configured_state", "configured"},
		},
		{
			name:   "last after duplicate wins over earlier entry",
			before: []model.ToolSummary{{ID: "gh", Installed: true}},
			after: []model.ToolSummary{
				{ID: "gh", Installed: true},
				{ID: "gh", Installed: false},
			},
			wantFields: []string{"installed"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := model.NewStatusReport(0)
			before.Tools = tt.before
			after := model.NewStatusReport(0)
			after.Tools = tt.after

			report := DiffSnapshots(before, after)

			if len(tt.wantFields) == 0 {
				if len(report.Changes) != 0 {
					t.Fatalf("expected no changes, got %#v", report.Changes)
				}
				return
			}
			if len(report.Changes) != 1 || !reflect.DeepEqual(report.Changes[0].Fields, tt.wantFields) {
				t.Fatalf("expected one change with fields %v, got %#v", tt.wantFields, report.Changes)
			}
		})
	}
}

func TestDiffSnapshotsReportsPerFieldChanges(t *testing.T) {
	base := func() model.ToolSummary {
		return model.ToolSummary{
			ID:              "gh",
			DisplayName:     "GitHub CLI",
			Category:        "vcs",
			Installed:       true,
			InstallPath:     "/usr/local/bin/gh",
			ConfiguredState: model.ConfiguredYes,
			Configured:      true,
			Capabilities:    model.Capability{HasContexts: true, CanSwitch: true},
			Current:         map[string]string{"account": "oldwinter"},
			Warnings:        []string{"token expires soon"},
			Errors:          []string{"rate limit seen earlier"},
		}
	}
	tests := []struct {
		name   string
		mutate func(*model.ToolSummary)
		fields []string
	}{
		{name: "display_name", mutate: func(s *model.ToolSummary) { s.DisplayName = "gh" }, fields: []string{"display_name"}},
		{name: "category", mutate: func(s *model.ToolSummary) { s.Category = "code-host" }, fields: []string{"category"}},
		{name: "installed", mutate: func(s *model.ToolSummary) { s.Installed = false }, fields: []string{"installed"}},
		{name: "install_path", mutate: func(s *model.ToolSummary) { s.InstallPath = "/opt/homebrew/bin/gh" }, fields: []string{"install_path"}},
		{name: "configured_state", mutate: func(s *model.ToolSummary) { s.ConfiguredState = model.ConfiguredNo }, fields: []string{"configured_state"}},
		{name: "configured", mutate: func(s *model.ToolSummary) { s.Configured = false }, fields: []string{"configured"}},
		{name: "capabilities", mutate: func(s *model.ToolSummary) { s.Capabilities.CanSwitch = false }, fields: []string{"capabilities"}},
		{name: "current", mutate: func(s *model.ToolSummary) { s.Current["account"] = "someone-else" }, fields: []string{"current"}},
		{name: "warnings", mutate: func(s *model.ToolSummary) { s.Warnings = []string{"different warning"} }, fields: []string{"warnings"}},
		{name: "errors", mutate: func(s *model.ToolSummary) { s.Errors = []string{"different error"} }, fields: []string{"errors"}},
		{
			name: "multiple fields keep comparison order",
			mutate: func(s *model.ToolSummary) {
				s.Installed = false
				s.InstallPath = ""
				s.ConfiguredState = model.ConfiguredNo
			},
			fields: []string{"installed", "install_path", "configured_state"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := model.NewStatusReport(0)
			before.Tools = []model.ToolSummary{base()}
			afterTool := base()
			tt.mutate(&afterTool)
			after := model.NewStatusReport(0)
			after.Tools = []model.ToolSummary{afterTool}

			report := DiffSnapshots(before, after)

			if report.Summary != (model.SnapshotDiffSummary{Changed: 1}) || len(report.Changes) != 1 {
				t.Fatalf("report = %#v, want exactly one changed tool", report)
			}
			change := report.Changes[0]
			if change.ToolID != "gh" || change.ChangeType != model.SnapshotChangeChanged {
				t.Fatalf("unexpected change record: %#v", change)
			}
			if !reflect.DeepEqual(change.Fields, tt.fields) {
				t.Fatalf("fields = %v, want %v", change.Fields, tt.fields)
			}
			if change.Before == nil || change.After == nil {
				t.Fatalf("changed entries should carry before and after snapshots, got %#v", change)
			}
		})
	}
}
