package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
	"github.com/oldwinter/all-cli/internal/tools"
	"github.com/spf13/cobra"
)

type cliFakeRunner struct {
	results map[string]execx.CmdResult
}

func (f cliFakeRunner) Run(_ context.Context, name string, args ...string) execx.CmdResult {
	key := name
	if len(args) > 0 {
		key += " " + strings.Join(args, " ")
	}
	if res, ok := f.results[key]; ok {
		return res
	}
	return execx.CmdResult{
		ExitCode: 1,
		Err:      errors.New("unexpected command"),
		Stderr:   "unexpected command",
	}
}

func executeTestCommand(t *testing.T, cmd *cobra.Command, args ...string) (string, string, error) {
	t.Helper()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)

	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

func stubToolEvaluation(t *testing.T, wantID string, summary model.ToolSummary) {
	t.Helper()

	oldFind := findToolByID
	oldEvaluate := evaluateToolSummary

	findToolByID = func(id string) (tools.ToolDefinition, bool) {
		if id != wantID {
			return tools.ToolDefinition{}, false
		}
		return tools.ToolDefinition{
			ID:           summary.ID,
			DisplayName:  summary.DisplayName,
			Category:     summary.Category,
			Binary:       wantID,
			Capabilities: summary.Capabilities,
		}, true
	}
	evaluateToolSummary = func(_ context.Context, def tools.ToolDefinition, _ execx.Runner) model.ToolSummary {
		if def.ID != wantID {
			t.Fatalf("unexpected tool definition: %#v", def)
		}
		return summary
	}

	t.Cleanup(func() {
		findToolByID = oldFind
		evaluateToolSummary = oldEvaluate
	})
}

func stubStatusRegistry(t *testing.T, defs []tools.ToolDefinition) {
	t.Helper()

	oldRegistry := defaultRegistry
	defaultRegistry = func() []tools.ToolDefinition {
		out := make([]tools.ToolDefinition, len(defs))
		copy(out, defs)
		return out
	}
	t.Cleanup(func() {
		defaultRegistry = oldRegistry
	})
}

func stubShowStatusSpinner(t *testing.T, show bool) {
	t.Helper()

	oldShow := showStatusSpinner
	showStatusSpinner = func() bool { return show }
	t.Cleanup(func() {
		showStatusSpinner = oldShow
	})
}
