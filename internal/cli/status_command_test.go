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

func TestStatusCommandJSONInstalledOnlyAndQuiet(t *testing.T) {
	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "aws", DisplayName: "AWS CLI", Category: "cloud", Binary: "aws"},
		{ID: "kubectl", DisplayName: "kubectl", Category: "k8s", Binary: "kubectl"},
		{ID: "gh", DisplayName: "gh", Category: "code", Binary: "gh"},
	})
	oldEvaluate := evaluateToolSummary
	evaluateToolSummary = func(_ context.Context, def tools.ToolDefinition, _ execx.Runner) model.ToolSummary {
		switch def.ID {
		case "aws":
			return model.ToolSummary{ID: "aws", Category: "cloud", Installed: true, ConfiguredState: model.ConfiguredYes, Configured: true}
		case "kubectl":
			return model.ToolSummary{ID: "kubectl", Category: "k8s", Installed: true, ConfiguredState: model.ConfiguredNo}
		case "gh":
			return model.ToolSummary{ID: "gh", Category: "code", Installed: false, ConfiguredState: model.ConfiguredUnknown}
		default:
			t.Fatalf("unexpected tool %s", def.ID)
			return model.ToolSummary{}
		}
	}
	t.Cleanup(func() { evaluateToolSummary = oldEvaluate })
	stubShowStatusSpinner(t, false)

	opts := &rootOptions{JSON: true, Timeout: time.Second}
	stdout, stderr, err := executeTestCommand(t, newStatusCommand(opts, cliFakeRunner{}), "--installed-only", "--quiet")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var got model.StatusReport
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if len(got.Tools) != 1 || got.Tools[0].ID != "kubectl" {
		t.Fatalf("unexpected tools payload: %#v", got.Tools)
	}
	if got.Legend.Installed == "" || got.GeneratedAt.IsZero() {
		t.Fatalf("expected legend and generated_at, got %#v", got)
	}
}

func TestStatusCommandJSONMissingOnly(t *testing.T) {
	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "aws", DisplayName: "AWS CLI", Category: "cloud", Binary: "aws"},
		{ID: "kubectl", DisplayName: "kubectl", Category: "k8s", Binary: "kubectl"},
		{ID: "gh", DisplayName: "gh", Category: "code", Binary: "gh"},
	})
	oldEvaluate := evaluateToolSummary
	evaluateToolSummary = func(_ context.Context, def tools.ToolDefinition, _ execx.Runner) model.ToolSummary {
		return model.ToolSummary{
			ID:              def.ID,
			Category:        def.Category,
			Installed:       def.ID != "gh",
			ConfiguredState: model.ConfiguredYes,
		}
	}
	t.Cleanup(func() { evaluateToolSummary = oldEvaluate })
	stubShowStatusSpinner(t, false)

	opts := &rootOptions{JSON: true, Timeout: time.Second}
	stdout, stderr, err := executeTestCommand(t, newStatusCommand(opts, cliFakeRunner{}), "--missing-only")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var got model.StatusReport
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if len(got.Tools) != 1 || got.Tools[0].ID != "gh" {
		t.Fatalf("unexpected tools payload: %#v", got.Tools)
	}
}

func TestStatusCommandRejectsConflictingInstallationFilters(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	_, _, err := executeTestCommand(t, newStatusCommand(opts, cliFakeRunner{}), "--installed-only", "--missing-only")
	if err == nil || !strings.Contains(err.Error(), "if any flags in the group") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStatusCommandPlainToolsFilterAndSpinner(t *testing.T) {
	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "aws", DisplayName: "AWS CLI", Category: "cloud", Binary: "aws"},
		{ID: "kubectl", DisplayName: "kubectl", Category: "k8s", Binary: "kubectl"},
	})
	oldEvaluate := evaluateToolSummary
	evaluateToolSummary = func(_ context.Context, def tools.ToolDefinition, _ execx.Runner) model.ToolSummary {
		return model.ToolSummary{ID: def.ID, Category: def.Category, Installed: true, ConfiguredState: model.ConfiguredYes, Configured: true}
	}
	t.Cleanup(func() { evaluateToolSummary = oldEvaluate })
	stubShowStatusSpinner(t, true)

	opts := &rootOptions{Timeout: time.Second}
	stdout, stderr, err := executeTestCommand(t, newStatusCommand(opts, cliFakeRunner{}), "--tools", "kubectl", "--group-by", "none")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "kubectl") || strings.Contains(stdout, "aws") {
		t.Fatalf("unexpected stdout:\n%s", stdout)
	}
	if !strings.Contains(stderr, "\r\033[K") {
		t.Fatalf("expected spinner cleanup in stderr, got %q", stderr)
	}
}

func TestStatusCommandJSONIncludesAdditiveDiagnostics(t *testing.T) {
	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "aws", DisplayName: "AWS CLI", Category: "cloud", Binary: "aws"},
	})
	oldEvaluate := evaluateToolSummary
	evaluateToolSummary = func(_ context.Context, def tools.ToolDefinition, _ execx.Runner) model.ToolSummary {
		return model.ToolSummary{
			ID:              def.ID,
			DisplayName:     def.DisplayName,
			Category:        def.Category,
			Installed:       true,
			ConfiguredState: model.ConfiguredUnknown,
			Errors:          []string{"context deadline exceeded"},
		}
	}
	t.Cleanup(func() { evaluateToolSummary = oldEvaluate })
	stubShowStatusSpinner(t, false)

	opts := &rootOptions{JSON: true, Timeout: time.Second}
	stdout, _, err := executeTestCommand(t, newStatusCommand(opts, cliFakeRunner{}), "--tools", "aws")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got model.StatusReport
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if len(got.Diagnostics) != 1 || got.Diagnostics[0].RelatedTool != "aws" {
		t.Fatalf("expected aws diagnostic in status payload, got %#v", got.Diagnostics)
	}
}

func TestStatusCommandErrorsOnInvalidGroupBy(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	_, _, err := executeTestCommand(t, newStatusCommand(opts, cliFakeRunner{}), "--group-by", "bad")
	if err == nil || !strings.Contains(err.Error(), "invalid --group-by value") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStatusCommandErrorsOnInvalidSort(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	_, _, err := executeTestCommand(t, newStatusCommand(opts, cliFakeRunner{}), "--sort", "bad")
	if err == nil || !strings.Contains(err.Error(), "invalid --sort value") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStatusHelpIncludesDescription(t *testing.T) {
	t.Parallel()

	cmd := newStatusCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{})
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("help: %v", err)
	}
	help := out.String()
	if !strings.Contains(help, "registry") {
		t.Fatalf("expected status long help to mention registry, got:\n%s", help)
	}
}

func TestStatusCommandErrorsOnUnknownToolsFilter(t *testing.T) {
	stubShowStatusSpinner(t, false)
	opts := &rootOptions{Timeout: time.Second}
	_, _, err := executeTestCommand(t, newStatusCommand(opts, cliFakeRunner{}), "--tools", "does-not-exist")
	if err == nil || !strings.Contains(err.Error(), `unknown tool ID "does-not-exist"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEvaluateStatusSummaryErrorsWhenToolMissing(t *testing.T) {
	oldFind := findToolByID
	findToolByID = func(id string) (tools.ToolDefinition, bool) {
		return tools.ToolDefinition{}, false
	}
	defer func() { findToolByID = oldFind }()

	_, err := evaluateStatusSummary(context.Background(), "missing", cliFakeRunner{}, time.Second)
	if err == nil || !strings.Contains(err.Error(), "missing tool definition not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunSingleToolStatusCommandErrorsWhenToolMissing(t *testing.T) {
	oldFind := findToolByID
	findToolByID = func(id string) (tools.ToolDefinition, bool) {
		return tools.ToolDefinition{}, false
	}
	defer func() { findToolByID = oldFind }()

	cmd := newStatusCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{})
	var stdout strings.Builder
	cmd.SetOut(&stdout)

	err := runSingleToolStatusCommand(cmd, &rootOptions{Timeout: time.Second}, cliFakeRunner{}, "missing")
	if err == nil || !strings.Contains(err.Error(), "missing tool definition not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}
