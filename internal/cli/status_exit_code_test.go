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

func stubStatusExitCodeRegistry(t *testing.T) {
	t.Helper()

	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "aws", DisplayName: "AWS CLI", Category: "cloud", Binary: "aws"},
		{ID: "gh", DisplayName: "gh", Category: "code", Binary: "gh"},
	})
	oldEvaluate := evaluateToolSummary
	evaluateToolSummary = func(_ context.Context, def tools.ToolDefinition, _ execx.Runner) model.ToolSummary {
		switch def.ID {
		case "aws":
			return model.ToolSummary{ID: "aws", Category: "cloud", Installed: true, ConfiguredState: model.ConfiguredYes, Configured: true}
		case "gh":
			return model.ToolSummary{ID: "gh", Category: "code", Installed: false, ConfiguredState: model.ConfiguredUnknown}
		default:
			t.Fatalf("unexpected tool %s", def.ID)
			return model.ToolSummary{}
		}
	}
	t.Cleanup(func() { evaluateToolSummary = oldEvaluate })
	stubShowStatusSpinner(t, false)
}

func TestStatusCommandExitCodeFailsAfterPrintingSummary(t *testing.T) {
	stubStatusExitCodeRegistry(t)

	stdout, stderr, err := executeTestCommand(t, newStatusCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}), "--summary-only", "--exit-code")

	if !errors.Is(err, errStatusNeedsAttention) {
		t.Fatalf("error = %v, want status needs attention", err)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if want := "Summary: 2 tools, 1 installed, 1 need attention\n"; stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
}

func TestStatusCommandExitCodeSucceedsWhenDisplayedToolsAreHealthy(t *testing.T) {
	stubStatusExitCodeRegistry(t)

	stdout, stderr, err := executeTestCommand(t, newStatusCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}), "--tools", "aws", "--summary-only", "--exit-code")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if want := "Summary: 1 tool, 1 installed, 0 need attention\n"; stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
}

func TestStatusCommandExitCodeUsesFilteredTools(t *testing.T) {
	stubStatusExitCodeRegistry(t)

	stdout, stderr, err := executeTestCommand(t, newStatusCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}), "--installed-only", "--summary-only", "--exit-code")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if want := "Summary: 1 tool, 1 installed, 0 need attention\n"; stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
}

func TestStatusCommandExitCodeKeepsJSONReport(t *testing.T) {
	stubStatusExitCodeRegistry(t)

	opts := &rootOptions{JSON: true, Timeout: time.Second}
	stdout, stderr, err := executeTestCommand(t, newStatusCommand(opts, cliFakeRunner{}), "--exit-code")

	if !errors.Is(err, errStatusNeedsAttention) {
		t.Fatalf("error = %v, want status needs attention", err)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	var got model.StatusReport
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if len(got.Tools) != 2 {
		t.Fatalf("expected full JSON report, got %#v", got.Tools)
	}
}

func TestStatusCommandExitCodeKeepsIDs(t *testing.T) {
	stubStatusExitCodeRegistry(t)

	stdout, stderr, err := executeTestCommand(t, newStatusCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}), "--ids", "--exit-code")

	if !errors.Is(err, errStatusNeedsAttention) {
		t.Fatalf("error = %v, want status needs attention", err)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if want := "aws\ngh\n"; stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
}

func TestStatusHelpDocumentsExitCode(t *testing.T) {
	t.Parallel()

	cmd := newStatusCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{})
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("help: %v", err)
	}
	if !strings.Contains(out.String(), "--exit-code") {
		t.Fatalf("expected status help to document --exit-code, got:\n%s", out.String())
	}
}
