package cli

import (
	"strings"

	diag "github.com/oldwinter/all-cli/internal/diagnose"
	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
	"github.com/oldwinter/all-cli/internal/output"
	"github.com/spf13/cobra"
)

func newReportCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	var toolsFilter string
	var categoriesFilter string
	var snapshotPath string

	cmd := &cobra.Command{
		Use:   "report",
		Short: "Create a shareable Markdown status report",
		Long: `Evaluates tracked tools and prints a Markdown report ready to paste into an
issue or pull request. Use --tools or --categories to limit external checks, or
--json to emit the existing machine-readable status report instead.

Use --from to render a saved JSON snapshot without probing local tools. The report
keeps the snapshot's timestamp and tool facts. Use --from - to read standard input
(limited to 1 MiB). Combine --from with --tools or --categories to include only
selected captured tools, preserving their original order and facts.`,
		Example: `  all-cli report
  all-cli report --tools kubectl,docker
  all-cli report --categories cloud,k8s
  all-cli report --tools gh --json
  all-cli report --from before.json
  all-cli report --from before.json --tools kubectl,docker
  all-cli report --from before.json --categories cloud
  all-cli snapshot --tools kubectl,docker --json | all-cli report --from -`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var report model.StatusReport
			var err error
			if cmd.Flags().Changed("from") {
				var selectedTools map[string]bool
				if strings.TrimSpace(toolsFilter) != "" {
					selectedTools, err = parseToolsFilter(toolsFilter)
					if err != nil {
						return err
					}
				}
				var selectedCategories map[string]bool
				selectedCategories, err = selectedReportCategories(categoriesFilter)
				if err != nil {
					return err
				}
				report, err = readStatusSnapshot(snapshotPath, cmd.InOrStdin())
				if err == nil {
					report.Tools = filterSnapshotTools(report.Tools, selectedTools)
					report.Tools = filterReportSnapshotCategories(report.Tools, selectedCategories)
				}
			} else {
				report, err = buildReportStatus(cmd.Context(), runner, opts.Timeout, toolsFilter, categoriesFilter)
			}
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
	cmd.Flags().StringVar(&categoriesFilter, "categories", "", "Comma-separated categories to include (e.g. cloud,k8s)")
	cmd.Flags().StringVar(&snapshotPath, "from", "", "Read a JSON snapshot file instead of probing tools (- for stdin)")
	return cmd
}
