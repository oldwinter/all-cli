package cli

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
	"github.com/oldwinter/all-cli/internal/tools"
)

func TestReportCommandPrintsMarkdownForSelectedTools(t *testing.T) {
	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "gh", DisplayName: "gh", Category: "code", Binary: "gh"},
		{ID: "kubectl", DisplayName: "kubectl", Category: "k8s", Binary: "kubectl"},
	})
	oldEvaluate := evaluateToolSummary
	evaluateToolSummary = func(_ context.Context, def tools.ToolDefinition, _ execx.Runner) model.ToolSummary {
		return model.ToolSummary{
			ID:              def.ID,
			Category:        def.Category,
			Installed:       true,
			ConfiguredState: model.ConfiguredYes,
			Current:         map[string]string{"user": "oldwinter"},
		}
	}
	t.Cleanup(func() { evaluateToolSummary = oldEvaluate })

	opts := &rootOptions{Timeout: time.Second}
	stdout, stderr, err := executeTestCommand(t, newReportCommand(opts, cliFakeRunner{}), "--tools", "gh")
	if err != nil {
		t.Fatalf("report: %v", err)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "# all-cli status report") || !strings.Contains(stdout, "| gh | code | yes | yes | user=oldwinter |") {
		t.Fatalf("unexpected report:\n%s", stdout)
	}
	if strings.Contains(stdout, "kubectl") {
		t.Fatalf("report ignored --tools filter:\n%s", stdout)
	}
}

func TestReportCommandJSONKeepsStatusContract(t *testing.T) {
	stubStatusRegistry(t, []tools.ToolDefinition{{
		ID: "aws", DisplayName: "AWS CLI", Category: "cloud", Binary: "aws",
	}})
	oldEvaluate := evaluateToolSummary
	evaluateToolSummary = func(_ context.Context, def tools.ToolDefinition, _ execx.Runner) model.ToolSummary {
		return model.ToolSummary{
			ID:              def.ID,
			Category:        def.Category,
			ConfiguredState: model.ConfiguredUnknown,
			Errors:          []string{"binary not found"},
		}
	}
	t.Cleanup(func() { evaluateToolSummary = oldEvaluate })

	opts := &rootOptions{JSON: true, Timeout: time.Second}
	stdout, _, err := executeTestCommand(t, newReportCommand(opts, cliFakeRunner{}), "--tools", "aws")
	if err != nil {
		t.Fatalf("report --json: %v", err)
	}
	var got model.StatusReport
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode report json: %v", err)
	}
	if got.SchemaVersion != model.SchemaVersionV01 || len(got.Tools) != 1 || got.Tools[0].ID != "aws" {
		t.Fatalf("unexpected report payload: %#v", got)
	}
	if len(got.Diagnostics) != 1 || got.Diagnostics[0].RelatedTool != "aws" {
		t.Fatalf("expected aws diagnostic, got %#v", got.Diagnostics)
	}
}

func TestRootCommandIncludesReport(t *testing.T) {
	root := NewRootCommand()
	report, _, err := root.Find([]string{"report"})
	if err != nil {
		t.Fatalf("find report command: %v", err)
	}
	if report.Name() != "report" || report.GroupID != "primary" {
		t.Fatalf("unexpected report registration: name=%q group=%q", report.Name(), report.GroupID)
	}
}
