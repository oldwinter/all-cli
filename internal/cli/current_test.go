package cli

import (
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
	"github.com/oldwinter/all-cli/internal/tools"
)

func TestCurrentCommandShowsCurrentContextsForInstalledContextTools(t *testing.T) {
	// Given
	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "aws", Category: "cloud", Binary: "aws", Capabilities: model.Capability{HasContexts: true}},
		{ID: "fd", Category: "navigation", Binary: "fd"},
		{ID: "kubectl", Category: "k8s", Binary: "kubectl", Capabilities: model.Capability{HasContexts: true}},
	})
	var evaluatedNonContext atomic.Bool
	oldEvaluate := evaluateToolSummary
	evaluateToolSummary = func(_ context.Context, def tools.ToolDefinition, _ execx.Runner) model.ToolSummary {
		switch def.ID {
		case "aws":
			return model.ToolSummary{
				ID:           "aws",
				Installed:    true,
				Capabilities: def.Capabilities,
				Current:      map[string]string{"region": "us-west-2", "profile": "work"},
			}
		case "fd":
			evaluatedNonContext.Store(true)
		case "kubectl":
			return model.ToolSummary{ID: "kubectl", Capabilities: def.Capabilities}
		}
		return model.ToolSummary{ID: def.ID, Capabilities: def.Capabilities}
	}
	t.Cleanup(func() { evaluateToolSummary = oldEvaluate })
	stubShowStatusSpinner(t, false)

	// When
	stdout, stderr, err := executeTestCommand(t, newCurrentCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}))

	// Then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if evaluatedNonContext.Load() {
		t.Fatal("expected tools without context support to be skipped")
	}
	if !strings.Contains(stdout, "TOOL  CURRENT") || !strings.Contains(stdout, "aws   profile=work region=us-west-2") {
		t.Fatalf("expected compact current overview, got:\n%s", stdout)
	}
	if strings.Contains(stdout, "fd") || strings.Contains(stdout, "kubectl") {
		t.Fatalf("expected non-context and uninstalled tools to be hidden, got:\n%s", stdout)
	}
}

func TestCurrentCommandIDsPrintsInstalledContextToolIDsOnly(t *testing.T) {
	// Given
	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "aws", Category: "cloud", Binary: "aws", Capabilities: model.Capability{HasContexts: true}},
		{ID: "docker", Category: "containers", Binary: "docker", Capabilities: model.Capability{HasContexts: true}},
		{ID: "fd", Category: "navigation", Binary: "fd"},
	})
	oldEvaluate := evaluateToolSummary
	evaluateToolSummary = func(_ context.Context, def tools.ToolDefinition, _ execx.Runner) model.ToolSummary {
		return model.ToolSummary{ID: def.ID, Installed: def.ID == "aws", Capabilities: def.Capabilities}
	}
	t.Cleanup(func() { evaluateToolSummary = oldEvaluate })
	stubShowStatusSpinner(t, true)

	// When
	stdout, stderr, err := executeTestCommand(t, newCurrentCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}), "--ids")

	// Then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != "aws\n" {
		t.Fatalf("expected installed context tool IDs only, got %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("expected --ids to suppress progress output, got %q", stderr)
	}
}

func TestCurrentCommandUnresolvedOnlyFiltersReadableContextsAcrossTextModes(t *testing.T) {
	// Given
	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "aws", Category: "cloud", Binary: "aws", Capabilities: model.Capability{HasContexts: true}},
		{ID: "docker", Category: "containers", Binary: "docker", Capabilities: model.Capability{HasContexts: true}},
		{ID: "kubectl", Category: "k8s", Binary: "kubectl", Capabilities: model.Capability{HasContexts: true}},
	})
	oldEvaluate := evaluateToolSummary
	evaluateToolSummary = func(_ context.Context, def tools.ToolDefinition, _ execx.Runner) model.ToolSummary {
		tool := model.ToolSummary{ID: def.ID, Installed: true, Capabilities: def.Capabilities}
		if def.ID == "aws" {
			tool.Current = map[string]string{"profile": "work"}
		}
		return tool
	}
	t.Cleanup(func() { evaluateToolSummary = oldEvaluate })
	stubShowStatusSpinner(t, false)

	// When / Then
	for _, tt := range []struct {
		name string
		args []string
		want string
	}{
		{name: "table", args: []string{"--unresolved-only"}, want: "docker   none\nkubectl  none"},
		{name: "ids", args: []string{"--unresolved-only", "--ids"}, want: "docker\nkubectl\n"},
		{name: "summary", args: []string{"--unresolved-only", "--summary-only"}, want: "Current: 2 tools, 0 with a readable context\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := executeTestCommand(t, newCurrentCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}), tt.args...)
			if err != nil || stderr != "" {
				t.Fatalf("current %v: stdout=%q stderr=%q err=%v", tt.args, stdout, stderr, err)
			}
			if !strings.Contains(stdout, tt.want) {
				t.Fatalf("current %v = %q, want %q", tt.args, stdout, tt.want)
			}
			if strings.Contains(stdout, "aws") {
				t.Fatalf("current %v included resolved tool: %q", tt.args, stdout)
			}
		})
	}
}

func TestCurrentCommandJSONPreservesStatusReportShape(t *testing.T) {
	// Given
	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "docker", Category: "containers", Binary: "docker", Capabilities: model.Capability{HasContexts: true, CanSwitch: true}},
	})
	oldEvaluate := evaluateToolSummary
	evaluateToolSummary = func(_ context.Context, def tools.ToolDefinition, _ execx.Runner) model.ToolSummary {
		return model.ToolSummary{
			ID:              def.ID,
			Category:        def.Category,
			Installed:       true,
			ConfiguredState: model.ConfiguredYes,
			Configured:      true,
			Capabilities:    def.Capabilities,
			Current:         map[string]string{"context": "desktop-linux"},
		}
	}
	t.Cleanup(func() { evaluateToolSummary = oldEvaluate })
	stubShowStatusSpinner(t, false)

	// When
	stdout, _, err := executeTestCommand(t, newCurrentCommand(&rootOptions{JSON: true, Timeout: time.Second}, cliFakeRunner{}))

	// Then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got model.StatusReport
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if got.SchemaVersion != model.SchemaVersionV01 || got.Legend.Current == "" || got.GeneratedAt.IsZero() {
		t.Fatalf("expected status report metadata, got %#v", got)
	}
	if len(got.Tools) != 1 || got.Tools[0].ID != "docker" || got.Tools[0].Current["context"] != "desktop-linux" {
		t.Fatalf("unexpected tools payload: %#v", got.Tools)
	}
}

func TestCurrentCommandSummaryOnlyPrintsOneLineAndSkipsSpinner(t *testing.T) {
	// Given
	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "aws", Category: "cloud", Binary: "aws", Capabilities: model.Capability{HasContexts: true}},
		{ID: "docker", Category: "containers", Binary: "docker", Capabilities: model.Capability{HasContexts: true}},
	})
	oldEvaluate := evaluateToolSummary
	evaluateToolSummary = func(_ context.Context, def tools.ToolDefinition, _ execx.Runner) model.ToolSummary {
		tool := model.ToolSummary{ID: def.ID, Installed: true, Capabilities: def.Capabilities}
		if def.ID == "aws" {
			tool.Current = map[string]string{"profile": "work"}
		}
		return tool
	}
	t.Cleanup(func() { evaluateToolSummary = oldEvaluate })
	stubShowStatusSpinner(t, true)

	// When
	stdout, stderr, err := executeTestCommand(t, newCurrentCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}), "--summary-only")

	// Then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := stdout, "Current: 2 tools, 1 with a readable context\n"; got != want {
		t.Fatalf("summary output = %q, want %q", got, want)
	}
	if stderr != "" {
		t.Fatalf("expected summary-only to skip spinner, got %q", stderr)
	}
}

func TestCurrentCommandMarkdownPrintsPasteReadyContextReport(t *testing.T) {
	// Given
	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "aws", Category: "cloud", Binary: "aws", Capabilities: model.Capability{HasContexts: true}},
		{ID: "docker", Category: "containers", Binary: "docker", Capabilities: model.Capability{HasContexts: true}},
	})
	oldEvaluate := evaluateToolSummary
	evaluateToolSummary = func(_ context.Context, def tools.ToolDefinition, _ execx.Runner) model.ToolSummary {
		tool := model.ToolSummary{ID: def.ID, Installed: true, Capabilities: def.Capabilities}
		if def.ID == "aws" {
			tool.Current = map[string]string{"profile": "work", "region": "us-west-2"}
		}
		return tool
	}
	t.Cleanup(func() { evaluateToolSummary = oldEvaluate })
	stubShowStatusSpinner(t, true)

	// When
	stdout, stderr, err := executeTestCommand(
		t,
		newCurrentCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}),
		"--markdown",
	)

	// Then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected markdown mode to skip spinner, got %q", stderr)
	}
	for _, want := range []string{
		"# all-cli current contexts",
		"| Tool | Current context |",
		"| aws | profile=work region=us-west-2 |",
		"| docker | none |",
		"Current: 2 tools, 1 with a readable context",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("markdown output missing %q:\n%s", want, stdout)
		}
	}
}

func TestCurrentCommandMarkdownRespectsUnresolvedOnly(t *testing.T) {
	// Given
	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "aws", Category: "cloud", Binary: "aws", Capabilities: model.Capability{HasContexts: true}},
		{ID: "docker", Category: "containers", Binary: "docker", Capabilities: model.Capability{HasContexts: true}},
	})
	oldEvaluate := evaluateToolSummary
	evaluateToolSummary = func(_ context.Context, def tools.ToolDefinition, _ execx.Runner) model.ToolSummary {
		tool := model.ToolSummary{ID: def.ID, Installed: true, Capabilities: def.Capabilities}
		if def.ID == "aws" {
			tool.Current = map[string]string{"profile": "work"}
		}
		return tool
	}
	t.Cleanup(func() { evaluateToolSummary = oldEvaluate })
	stubShowStatusSpinner(t, false)

	// When
	stdout, _, err := executeTestCommand(
		t,
		newCurrentCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}),
		"--unresolved-only",
		"--markdown",
	)

	// Then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "| docker | none |") || strings.Contains(stdout, "| aws |") {
		t.Fatalf("markdown did not preserve unresolved-only filter:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Current: 1 tools, 0 with a readable context") {
		t.Fatalf("unexpected filtered summary:\n%s", stdout)
	}
}

func TestCurrentCommandJSONTakesPrecedenceOverSummaryOnly(t *testing.T) {
	// Given
	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "docker", Category: "containers", Binary: "docker", Capabilities: model.Capability{HasContexts: true}},
	})
	oldEvaluate := evaluateToolSummary
	evaluateToolSummary = func(_ context.Context, def tools.ToolDefinition, _ execx.Runner) model.ToolSummary {
		return model.ToolSummary{ID: def.ID, Installed: true, Capabilities: def.Capabilities}
	}
	t.Cleanup(func() { evaluateToolSummary = oldEvaluate })
	stubShowStatusSpinner(t, false)

	// When
	stdout, _, err := executeTestCommand(t, newCurrentCommand(&rootOptions{JSON: true, Timeout: time.Second}, cliFakeRunner{}), "--summary-only")

	// Then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, `"tools"`) || strings.Contains(stdout, "Current:") {
		t.Fatalf("expected JSON to take precedence, got %q", stdout)
	}
}

func TestCurrentCommandJSONTakesPrecedenceOverMarkdown(t *testing.T) {
	// Given
	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "docker", Category: "containers", Binary: "docker", Capabilities: model.Capability{HasContexts: true}},
	})
	oldEvaluate := evaluateToolSummary
	evaluateToolSummary = func(_ context.Context, def tools.ToolDefinition, _ execx.Runner) model.ToolSummary {
		return model.ToolSummary{ID: def.ID, Installed: true, Capabilities: def.Capabilities}
	}
	t.Cleanup(func() { evaluateToolSummary = oldEvaluate })
	stubShowStatusSpinner(t, false)

	// When
	stdout, _, err := executeTestCommand(
		t,
		newCurrentCommand(&rootOptions{JSON: true, Timeout: time.Second}, cliFakeRunner{}),
		"--markdown",
	)

	// Then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, `"tools"`) || strings.Contains(stdout, "# all-cli current contexts") {
		t.Fatalf("expected JSON to take precedence, got %q", stdout)
	}
}

func TestCurrentCommandMarkdownConflictsWithCompactModes(t *testing.T) {
	for _, conflicting := range []string{"--ids", "--summary-only"} {
		t.Run(conflicting, func(t *testing.T) {
			_, _, err := executeTestCommand(
				t,
				newCurrentCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}),
				"--markdown",
				conflicting,
			)
			if err == nil || !strings.Contains(err.Error(), "if any flags in the group") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestCurrentCommandJSONTakesPrecedenceOverUnresolvedOnly(t *testing.T) {
	// Given
	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "aws", Category: "cloud", Binary: "aws", Capabilities: model.Capability{HasContexts: true}},
	})
	oldEvaluate := evaluateToolSummary
	evaluateToolSummary = func(_ context.Context, def tools.ToolDefinition, _ execx.Runner) model.ToolSummary {
		return model.ToolSummary{ID: def.ID, Installed: true, Capabilities: def.Capabilities, Current: map[string]string{"profile": "work"}}
	}
	t.Cleanup(func() { evaluateToolSummary = oldEvaluate })
	stubShowStatusSpinner(t, false)

	// When
	stdout, _, err := executeTestCommand(t, newCurrentCommand(&rootOptions{JSON: true, Timeout: time.Second}, cliFakeRunner{}), "--unresolved-only")

	// Then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got model.StatusReport
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if len(got.Tools) != 1 || got.Tools[0].ID != "aws" {
		t.Fatalf("expected JSON to preserve resolved tools, got %#v", got.Tools)
	}
}

func TestCurrentCommandUnresolvedOnlyExplainsEmptyResults(t *testing.T) {
	// Given
	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "aws", Category: "cloud", Binary: "aws", Capabilities: model.Capability{HasContexts: true}},
	})
	oldEvaluate := evaluateToolSummary
	evaluateToolSummary = func(_ context.Context, def tools.ToolDefinition, _ execx.Runner) model.ToolSummary {
		return model.ToolSummary{ID: def.ID, Installed: true, Capabilities: def.Capabilities, Current: map[string]string{"profile": "work"}}
	}
	t.Cleanup(func() { evaluateToolSummary = oldEvaluate })
	stubShowStatusSpinner(t, false)

	// When
	stdout, _, err := executeTestCommand(t, newCurrentCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}), "--unresolved-only")

	// Then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "without a readable current context") || strings.Contains(stdout, "No installed context-aware tools found") {
		t.Fatalf("unexpected empty unresolved output: %q", stdout)
	}
}

func TestCurrentCommandToolsFilterEvaluatesOnlySelectedContextTools(t *testing.T) {
	// Given
	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "aws", Category: "cloud", Binary: "aws", Capabilities: model.Capability{HasContexts: true}},
		{ID: "docker", Category: "containers", Binary: "docker", Capabilities: model.Capability{HasContexts: true}},
		{ID: "fd", Category: "navigation", Binary: "fd"},
	})
	evaluated := make(chan string, 3)
	oldEvaluate := evaluateToolSummary
	evaluateToolSummary = func(_ context.Context, def tools.ToolDefinition, _ execx.Runner) model.ToolSummary {
		evaluated <- def.ID
		return model.ToolSummary{
			ID:           def.ID,
			Installed:    true,
			Capabilities: def.Capabilities,
			Current:      map[string]string{"context": "desktop-linux"},
		}
	}
	t.Cleanup(func() { evaluateToolSummary = oldEvaluate })
	stubShowStatusSpinner(t, false)

	// When
	stdout, stderr, err := executeTestCommand(
		t,
		newCurrentCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}),
		"--tools", "docker",
	)

	// Then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if got := <-evaluated; got != "docker" {
		t.Fatalf("evaluated tool = %q, want docker", got)
	}
	select {
	case got := <-evaluated:
		t.Fatalf("unexpected additional tool evaluation: %q", got)
	default:
	}
	if !strings.Contains(stdout, "docker  context=desktop-linux") || strings.Contains(stdout, "aws") {
		t.Fatalf("expected only docker current context, got:\n%s", stdout)
	}
}

func TestCurrentCommandCategoriesFilterEvaluatesOnlyMatchingContextTools(t *testing.T) {
	// Given
	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "aws", Category: "cloud", Binary: "aws", Capabilities: model.Capability{HasContexts: true}},
		{ID: "docker", Category: "containers", Binary: "docker", Capabilities: model.Capability{HasContexts: true}},
		{ID: "kubectl", Category: "k8s", Binary: "kubectl", Capabilities: model.Capability{HasContexts: true}},
	})
	evaluated := make(chan string, 3)
	oldEvaluate := evaluateToolSummary
	evaluateToolSummary = func(_ context.Context, def tools.ToolDefinition, _ execx.Runner) model.ToolSummary {
		evaluated <- def.ID
		return model.ToolSummary{
			ID:           def.ID,
			Installed:    true,
			Capabilities: def.Capabilities,
			Current:      map[string]string{"profile": "work"},
		}
	}
	t.Cleanup(func() { evaluateToolSummary = oldEvaluate })
	stubShowStatusSpinner(t, false)

	// When
	stdout, stderr, err := executeTestCommand(
		t,
		newCurrentCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}),
		"--tools", "aws,kubectl", "--categories", "cloud",
	)

	// Then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if got := <-evaluated; got != "aws" {
		t.Fatalf("evaluated tool = %q, want aws", got)
	}
	select {
	case got := <-evaluated:
		t.Fatalf("unexpected additional tool evaluation: %q", got)
	default:
	}
	if !strings.Contains(stdout, "aws   profile=work") || strings.Contains(stdout, "kubectl") {
		t.Fatalf("expected only cloud current context, got:\n%s", stdout)
	}
}

func TestCurrentCommandRejectsUnknownCategoriesBeforeEvaluation(t *testing.T) {
	// Given
	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "aws", Category: "cloud", Binary: "aws", Capabilities: model.Capability{HasContexts: true}},
	})
	oldEvaluate := evaluateToolSummary
	evaluateToolSummary = func(_ context.Context, _ tools.ToolDefinition, _ execx.Runner) model.ToolSummary {
		t.Fatal("unexpected tool evaluation")
		return model.ToolSummary{}
	}
	t.Cleanup(func() { evaluateToolSummary = oldEvaluate })
	stubShowStatusSpinner(t, false)

	// When
	_, _, err := executeTestCommand(
		t,
		newCurrentCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}),
		"--categories", "workflows",
	)

	// Then
	if err == nil || !strings.Contains(err.Error(), "unknown categories: workflows") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCurrentCommandCompletesCategoryFilters(t *testing.T) {
	// Given
	root := NewRootCommand()

	// When
	stdout, _, err := executeTestCommand(t, root, "__complete", "current", "--categories", "cl")

	// Then
	if err != nil {
		t.Fatalf("complete current --categories: %v", err)
	}
	if !strings.Contains(stdout, "cloud") {
		t.Fatalf("cloud completion missing: %q", stdout)
	}
}

func TestRootRegistersCurrentAsPrimaryCommand(t *testing.T) {
	// Given
	root := NewRootCommand()

	// When
	current, _, err := root.Find([]string{"current"})

	// Then
	if err != nil {
		t.Fatalf("find current command: %v", err)
	}
	if current.Name() != "current" || current.GroupID != "primary" {
		t.Fatalf("unexpected current command registration: name=%q group=%q", current.Name(), current.GroupID)
	}
}
