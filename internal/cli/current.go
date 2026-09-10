package cli

import (
	"github.com/oldwinter/all-cli/internal/diagnose"
	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/output"
	"github.com/oldwinter/all-cli/internal/tools"
	"github.com/spf13/cobra"
)

func newCurrentCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	var toolsFilter string
	var categoriesFilter string

	cmd := &cobra.Command{
		Use:   "current",
		Short: "Show current contexts across installed CLI tools",
		Long: `Shows one compact view of the active accounts, clusters, projects, and
environments reported by installed tools that expose context-like state.

Use --tools or --categories to evaluate only selected tools and skip unrelated
	external commands. When combined, both filters must match.`,
		Example: `  all-cli current
  all-cli current --tools kubectl,docker
  all-cli current --categories cloud,k8s
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
			if !opts.JSON && showStatusSpinner() {
				spinner = newProgressSpinner(cmd.ErrOrStderr(), len(contextRegistry))
				spinner.Start()
			}

			report := evaluateStatusRegistry(cmd.Context(), contextRegistry, runner, opts.Timeout, spinner)
			if spinner != nil {
				spinner.Stop()
			}

			installed := report.Tools[:0]
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
			output.PrintCurrentTable(cmd.OutOrStdout(), report)
			return nil
		},
	}
	cmd.Flags().StringVar(&toolsFilter, "tools", "", "Comma-separated tool IDs to show (e.g. kubectl,docker)")
	cmd.Flags().StringVar(&categoriesFilter, "categories", "", "Comma-separated categories to show (e.g. cloud,k8s)")
	return cmd
}
