package cli

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
	"github.com/oldwinter/all-cli/internal/tools"
)

func TestCurrentCommandExitCodePassesWhenAllContextsAreReadable(t *testing.T) {
	stubCurrentContexts(t, map[string]map[string]string{
		"aws":    {"profile": "work"},
		"docker": {"context": "desktop-linux"},
	})

	stdout, stderr, err := executeTestCommand(
		t,
		newCurrentCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}),
		"--summary-only", "--exit-code",
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := stdout, "Current: 2 tools, 2 with a readable context\n"; got != want {
		t.Fatalf("summary output = %q, want %q", got, want)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
}

func TestCurrentCommandExitCodeFailsWhenAnyContextIsUnresolved(t *testing.T) {
	stubCurrentContexts(t, map[string]map[string]string{
		"aws":    {"profile": "work"},
		"docker": nil,
	})

	stdout, stderr, err := executeTestCommand(
		t,
		newCurrentCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}),
		"--summary-only", "--exit-code",
	)

	if !errors.Is(err, errCurrentContextsUnresolved) {
		t.Fatalf("error = %v, want errCurrentContextsUnresolved", err)
	}
	if got, want := stdout, "Current: 2 tools, 1 with a readable context\n"; got != want {
		t.Fatalf("summary output = %q, want %q", got, want)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
}

func TestCurrentCommandExitCodeComposesWithUnresolvedOnlyAndIDs(t *testing.T) {
	stubCurrentContexts(t, map[string]map[string]string{
		"aws":    {"profile": "work"},
		"docker": nil,
	})

	stdout, stderr, err := executeTestCommand(
		t,
		newCurrentCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}),
		"--unresolved-only", "--ids", "--exit-code",
	)

	if !errors.Is(err, errCurrentContextsUnresolved) {
		t.Fatalf("error = %v, want errCurrentContextsUnresolved", err)
	}
	if got, want := stdout, "docker\n"; got != want {
		t.Fatalf("ids output = %q, want %q", got, want)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
}

func TestCurrentCommandExitCodeIgnoresUninstalledTools(t *testing.T) {
	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "aws", Category: "cloud", Binary: "aws", Capabilities: model.Capability{HasContexts: true}},
		{ID: "docker", Category: "containers", Binary: "docker", Capabilities: model.Capability{HasContexts: true}},
	})
	oldEvaluate := evaluateToolSummary
	evaluateToolSummary = func(_ context.Context, def tools.ToolDefinition, _ execx.Runner) model.ToolSummary {
		return model.ToolSummary{
			ID:           def.ID,
			Installed:    def.ID == "aws",
			Capabilities: def.Capabilities,
			Current:      map[string]string{"profile": "work"},
		}
	}
	t.Cleanup(func() { evaluateToolSummary = oldEvaluate })
	stubShowStatusSpinner(t, false)

	stdout, stderr, err := executeTestCommand(
		t,
		newCurrentCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}),
		"--exit-code",
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "aws") || strings.Contains(stdout, "docker") {
		t.Fatalf("expected only the installed tool, got:\n%s", stdout)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
}

func TestCurrentCommandExitCodeJSONKeepsFullReport(t *testing.T) {
	stubCurrentContexts(t, map[string]map[string]string{
		"aws":    {"profile": "work"},
		"docker": nil,
	})

	stdout, stderr, err := executeTestCommand(
		t,
		newCurrentCommand(&rootOptions{JSON: true, Timeout: time.Second}, cliFakeRunner{}),
		"--unresolved-only", "--exit-code",
	)

	if !errors.Is(err, errCurrentContextsUnresolved) {
		t.Fatalf("error = %v, want errCurrentContextsUnresolved", err)
	}
	var report model.StatusReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if len(report.Tools) != 2 {
		t.Fatalf("expected JSON to keep the full installed report, got %#v", report.Tools)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
}

func TestCurrentCommandUnresolvedOnlyTreatsBlankValuesAsUnresolved(t *testing.T) {
	stubCurrentContexts(t, map[string]map[string]string{
		"docker": {"context": "   "},
	})

	stdout, stderr, err := executeTestCommand(
		t,
		newCurrentCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}),
		"--unresolved-only", "--ids", "--exit-code",
	)

	if !errors.Is(err, errCurrentContextsUnresolved) {
		t.Fatalf("error = %v, want errCurrentContextsUnresolved", err)
	}
	if got, want := stdout, "docker\n"; got != want {
		t.Fatalf("ids output = %q, want %q", got, want)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
}

func stubCurrentContexts(t *testing.T, contexts map[string]map[string]string) {
	t.Helper()

	defs := make([]tools.ToolDefinition, 0, len(contexts))
	for id := range contexts {
		defs = append(defs, tools.ToolDefinition{
			ID:           id,
			Category:     "test",
			Binary:       id,
			Capabilities: model.Capability{HasContexts: true},
		})
	}
	stubStatusRegistry(t, defs)

	oldEvaluate := evaluateToolSummary
	evaluateToolSummary = func(_ context.Context, def tools.ToolDefinition, _ execx.Runner) model.ToolSummary {
		return model.ToolSummary{
			ID:           def.ID,
			Installed:    true,
			Capabilities: def.Capabilities,
			Current:      contexts[def.ID],
		}
	}
	t.Cleanup(func() { evaluateToolSummary = oldEvaluate })
	stubShowStatusSpinner(t, false)
}
