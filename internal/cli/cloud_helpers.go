package cli

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	diag "github.com/oldwinter/all-cli/internal/diagnose"
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

func evaluateStatusSummary(ctx context.Context, toolID string, runner execx.Runner, timeouts timeoutSetting) (model.ToolSummary, error) {
	def, ok := findToolByID(toolID)
	if !ok {
		return model.ToolSummary{}, fmt.Errorf("%s tool definition not found", toolID)
	}
	return evaluateToolSummary(ctx, def, runnerForTool(runner, timeouts, def)), nil
}

// singleToolStatusReport wraps one tool summary in the same StatusReport shape
// that `status --json` emits. Per-tool output therefore carries schema_version,
// generated_at, and the legend, so it satisfies schemas/status-report-v0.1.json
// and can be piped into `all-cli diff`.
func singleToolStatusReport(summary model.ToolSummary) model.StatusReport {
	report := model.NewStatusReport(1)
	report.GeneratedAt = time.Now()
	report.Tools[0] = summary
	return report
}

func printSingleToolStatus(w io.Writer, summary model.ToolSummary) {
	output.PrintStatusTable(w, singleToolStatusReport(summary))
}

func runSingleToolStatusCommand(cmd *cobra.Command, opts *rootOptions, runner execx.Runner, toolID string) error {
	summary, err := evaluateStatusSummary(cmd.Context(), toolID, runner, opts.timeouts())
	if err != nil {
		return err
	}
	if opts.JSON {
		report := singleToolStatusReport(summary)
		report.Diagnostics = diag.Generate(report, diag.Options{Profile: diag.ProfileAgent}).Diagnostics
		return output.PrintJSON(cmd.OutOrStdout(), report)
	}
	printSingleToolStatus(cmd.OutOrStdout(), summary)
	return nil
}

// reportUseFailure reports a failed context switch once. With --json the
// machine-readable failure envelope goes to stdout and the human-readable
// "Error: ..." line on stderr is suppressed, mirroring `diff --exit-code`.
// Without it the same failure was printed twice, in two formats.
func reportUseFailure(cmd *cobra.Command, opts *rootOptions, toolID string, err error) error {
	if opts.JSON {
		_ = output.PrintJSON(cmd.OutOrStdout(), model.UseResult{OK: false, ToolID: toolID, Error: err.Error()})
		cmd.Root().SilenceErrors = true
	}
	return err
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
