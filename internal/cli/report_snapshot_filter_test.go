package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
	"github.com/oldwinter/all-cli/internal/tools"
)

func TestReportFromSnapshotFiltersTools(t *testing.T) {
	report := model.NewStatusReport(3)
	report.GeneratedAt = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	report.Diagnostics = []model.DiagnosticItem{{
		ID: "stale", RelatedTool: "stale-tool", Problem: "stale snapshot diagnostic",
	}}
	report.Tools = []model.ToolSummary{
		{ID: "gh", Category: "code", Installed: true, ConfiguredState: model.ConfiguredYes,
			Current: map[string]string{"user": "saved-user"}, Warnings: []string{"saved gh warning"}},
		{ID: "kubectl", Category: "k8s", Installed: true, ConfiguredState: model.ConfiguredYes,
			Current: map[string]string{"context": "saved|cluster"}, Errors: []string{"saved kubectl error"}},
		{ID: "aws", Category: "cloud", ConfiguredState: model.ConfiguredUnknown},
	}
	path := writeStatusReportFixture(t, t.TempDir(), "snapshot.json", report)
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	oldEvaluate := evaluateToolSummary
	evaluateToolSummary = func(_ context.Context, _ tools.ToolDefinition, _ execx.Runner) model.ToolSummary {
		t.Error("filtering a snapshot must not evaluate local tools")
		return model.ToolSummary{}
	}
	t.Cleanup(func() { evaluateToolSummary = oldEvaluate })

	for _, tt := range []struct {
		name   string
		filter string
		want   []model.ToolSummary
	}{
		{"single", "kubectl", report.Tools[1:2]},
		{"present and absent", "kubectl,fd", report.Tools[1:2]},
		{"snapshot order and duplicates", " aws, gh,gh, ", []model.ToolSummary{report.Tools[0], report.Tools[2]}},
		{"absent tracked tool", "fd", []model.ToolSummary{}},
		{"empty keeps all", "", report.Tools},
		{"whitespace keeps all", "  ", report.Tools},
	} {
		for _, source := range []string{path, "-"} {
			for _, asJSON := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/json=%t", tt.name, source, asJSON), func(t *testing.T) {
					cmd := newReportCommand(&rootOptions{JSON: asJSON}, cliFakeRunner{})
					cmd.SetIn(strings.NewReader(string(data)))
					stdout, stderr, err := executeTestCommand(t, cmd, "--from", source, "--tools", tt.filter)
					if err != nil || stderr != "" {
						t.Fatalf("report: err=%v stderr=%q", err, stderr)
					}
					if asJSON {
						assertFilteredReportJSON(t, stdout, report, tt.want)
						return
					}
					assertFilteredReportMarkdown(t, stdout, report.Tools, tt.want)
				})
			}
		}
	}
}

func assertFilteredReportJSON(t *testing.T, stdout string, original model.StatusReport, want []model.ToolSummary) {
	t.Helper()
	var got model.StatusReport
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Tools, want) || !got.GeneratedAt.Equal(original.GeneratedAt) ||
		got.SchemaVersion != original.SchemaVersion || !reflect.DeepEqual(got.Legend, original.Legend) {
		t.Fatalf("captured facts or tool selection changed: %#v", got)
	}
	selected := make(map[string]bool, len(want))
	for _, tool := range want {
		selected[tool.ID] = true
	}
	if len(want) > 0 && len(got.Diagnostics) == 0 {
		t.Fatal("selected tool issues must produce diagnostics")
	}
	for _, diagnostic := range got.Diagnostics {
		if !selected[diagnostic.RelatedTool] {
			t.Errorf("diagnostic for excluded tool: %#v", diagnostic)
		}
	}
}

func assertFilteredReportMarkdown(t *testing.T, stdout string, all, want []model.ToolSummary) {
	t.Helper()
	if !strings.Contains(stdout, "Generated: `2026-01-02T03:04:05Z`") {
		t.Fatal("captured timestamp missing")
	}
	selected := make(map[string]bool, len(want))
	previous := -1
	for _, tool := range want {
		selected[tool.ID] = true
		position := strings.Index(stdout, "| "+tool.ID+" |")
		if position <= previous {
			t.Fatalf("missing or reordered tool %q:\n%s", tool.ID, stdout)
		}
		previous = position
	}
	for _, tool := range all {
		if strings.Contains(stdout, "| "+tool.ID+" |") != selected[tool.ID] {
			t.Errorf("unexpected selection for %q:\n%s", tool.ID, stdout)
		}
		for _, message := range append(tool.Warnings, tool.Errors...) {
			if strings.Contains(stdout, message) != selected[tool.ID] {
				t.Errorf("unexpected inclusion of message %q:\n%s", message, stdout)
			}
		}
	}
	if selected["kubectl"] && !strings.Contains(stdout, `context=saved\|cluster`) {
		t.Fatal("captured context or Markdown escaping changed")
	}
	if len(want) == 0 && !strings.Contains(stdout, "No tools in this report.") {
		t.Fatal("empty selection needs the empty report message")
	}
}
