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
