package cli

import (
	"fmt"

	"github.com/oldwinter/all-cli/internal/diagnose"
	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
	"github.com/oldwinter/all-cli/internal/output"
	"github.com/oldwinter/all-cli/internal/tools"
	"github.com/spf13/cobra"
)

func newCurrentCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	var toolsFilter string
	var categoriesFilter string
	var ids bool
	var summaryOnly bool

	cmd := &cobra.Command{
		Use:   "current",
		Short: "Show current contexts across installed CLI tools",
		Long: `Shows one compact view of the active accounts, clusters, projects, and
environments reported by installed tools that expose context-like state.

Use --tools or --categories to evaluate only selected tools and skip unrelated
external commands. When combined, both filters must match.
Use --ids to print one installed context-aware tool ID per line.
Use --summary-only to print a prompt-friendly count instead of the context table.`,
		Example: `  all-cli current
  all-cli current --tools kubectl,docker
  all-cli current --categories cloud,k8s
  all-cli current --ids
  all-cli current --summary-only
  all-cli current --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			registry, err := registryForToolsFilter(toolsFilter)
			if err != nil {
				return err
			}
			registry, err = registryForCategoriesFilter(registry, categoriesFilter)
			if err != nil {
				return err
			}
			contextRegistry := make([]tools.ToolDefinition, 0, len(registry))
			for _, definition := range registry {
				if definition.Capabilities.HasContexts {
					contextRegistry = append(contextRegistry, definition)
				}
			}

			var spinner *progressSpinner
			if !opts.JSON && !ids && !summaryOnly && !opts.NoProgress && showStatusSpinner() {
				spinner = newProgressSpinner(cmd.ErrOrStderr(), len(contextRegistry))
				spinner.Start()
			}

			report := evaluateStatusRegistry(cmd.Context(), contextRegistry, runner, opts.timeouts(), spinner)
			if spinner != nil {
				spinner.Stop()
			}

			installed := make([]model.ToolSummary, 0, len(report.Tools))
			for _, tool := range report.Tools {
				if tool.Installed {
					installed = append(installed, tool)
				}
			}
			report.Tools = installed
			sortToolSummaries(report.Tools, statusSortTool)

			if opts.JSON {
				report.Diagnostics = diagnose.Generate(report, diagnose.Options{Profile: diagnose.ProfileAgent}).Diagnostics
				return output.PrintJSON(cmd.OutOrStdout(), report)
			}
			if ids {
				return printCurrentIDs(cmd, report.Tools)
			}
			if summaryOnly {
				output.PrintCurrentSummary(cmd.OutOrStdout(), report)
				return nil
			}
			output.PrintCurrentTable(cmd.OutOrStdout(), report)
			return nil
		},
	}
	cmd.Flags().StringVar(&toolsFilter, "tools", "", "Comma-separated tool IDs to show (e.g. kubectl,docker)")
	cmd.Flags().StringVar(&categoriesFilter, "categories", "", "Comma-separated categories to show (e.g. cloud,k8s)")
	cmd.Flags().BoolVar(&ids, "ids", false, "Print only installed context-aware tool IDs, one per line (ignored with --json)")
	cmd.Flags().BoolVar(&summaryOnly, "summary-only", false, "Print only the one-line current-context summary (ignored with --json)")
	return cmd
}

func printCurrentIDs(cmd *cobra.Command, toolsList []model.ToolSummary) error {
	for _, tool := range toolsList {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), tool.ID); err != nil {
			return err
		}
	}
	return nil
}
