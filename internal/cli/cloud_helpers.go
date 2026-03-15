package cli

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
	"github.com/oldwinter/all-cli/internal/output"
	"github.com/oldwinter/all-cli/internal/tools"
	"github.com/spf13/cobra"
)

var (
	findToolByID        = tools.FindByID
	evaluateToolSummary = tools.Evaluate
)

func evaluateStatusSummary(ctx context.Context, toolID string, runner execx.Runner, defaultTimeout time.Duration) (model.ToolSummary, error) {
	def, ok := findToolByID(toolID)
	if !ok {
		return model.ToolSummary{}, fmt.Errorf("%s tool definition not found", toolID)
	}
	return evaluateToolSummary(ctx, def, runnerForTool(runner, defaultTimeout, def)), nil
}

func printSingleToolStatus(w io.Writer, summary model.ToolSummary) {
	report := model.NewStatusReport(1)
	report.Tools[0] = summary
	output.PrintStatusTable(w, report)
}

func runSingleToolStatusCommand(cmd *cobra.Command, opts *rootOptions, runner execx.Runner, toolID string) error {
	summary, err := evaluateStatusSummary(cmd.Context(), toolID, runner, opts.Timeout)
	if err != nil {
		return err
	}
	if opts.JSON {
		return output.PrintJSON(cmd.OutOrStdout(), summary)
	}
	printSingleToolStatus(cmd.OutOrStdout(), summary)
	return nil
}

func printDiagnostics(w io.Writer, warnings, errs []string) {
	for _, warning := range dedupeMessages(warnings) {
		fmt.Fprintf(w, "warning: %s\n", warning)
	}
	for _, err := range dedupeMessages(errs) {
		fmt.Fprintf(w, "error: %s\n", err)
	}
}

func dedupeMessages(messages []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(messages))
	for _, msg := range messages {
		msg = strings.TrimSpace(msg)
		if msg == "" || seen[msg] {
			continue
		}
		seen[msg] = true
		out = append(out, msg)
	}
	return out
}
