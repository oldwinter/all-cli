package cli

import (
	diag "github.com/oldwinter/all-cli/internal/diagnose"
	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/output"
	"github.com/spf13/cobra"
)

func newReportCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	var toolsFilter string

	cmd := &cobra.Command{
		Use:   "report",
		Short: "Create a shareable Markdown status report",
		Long: `Evaluates tracked tools and prints a Markdown report ready to paste into an
issue or pull request. Use --tools to limit external checks or --json to emit the
existing machine-readable status report instead.`,
		Example: `  all-cli report
  all-cli report --tools kubectl,docker
  all-cli report --tools gh --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			report, err := buildStatusReport(cmd.Context(), runner, opts.Timeout, toolsFilter)
			if err != nil {
				return err
			}
			if opts.JSON {
				report.Diagnostics = diag.Generate(report, diag.Options{Profile: diag.ProfileAgent}).Diagnostics
				return output.PrintJSON(cmd.OutOrStdout(), report)
			}
			output.PrintStatusMarkdown(cmd.OutOrStdout(), report)
			return nil
		},
	}

	cmd.Flags().StringVar(&toolsFilter, "tools", "", "Comma-separated tool IDs to include")
	return cmd
}
