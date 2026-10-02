package cli

import (
	"fmt"
	"sort"

	"github.com/oldwinter/all-cli/internal/diagnose"
	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/output"
	"github.com/oldwinter/all-cli/internal/tools"
	"github.com/spf13/cobra"
)

// toolLookPath is the install probe used by the --ids fast path; tests stub it.
var toolLookPath = execx.LookPath

func newCurrentCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	var toolsFilter string
	var categoriesFilter string
	var ids bool

	cmd := &cobra.Command{
		Use:   "current",
		Short: "Show current contexts across installed CLI tools",
		Long: `Shows one compact view of the active accounts, clusters, projects, and
environments reported by installed tools that expose context-like state.

Use --tools or --categories to evaluate only selected tools and skip unrelated
external commands. When combined, both filters must match. Use --ids to print
one matching installed tool ID per line for shell pipelines; it resolves via
PATH lookup only and never invokes a tool command.`,
		Example: `  all-cli current
  all-cli current --tools kubectl,docker
  all-cli current --categories cloud,k8s
  all-cli current --ids | xargs -n1 all-cli describe
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

			// --ids only needs the installed tool set: a PATH lookup per
			// context-capable tool. Skipping evaluateStatusRegistry keeps a
			// slow or hung tool probe from delaying the ID listing.
			if ids && !opts.JSON {
				toolIDs := make([]string, 0, len(contextRegistry))
				for _, definition := range contextRegistry {
					if _, err := toolLookPath(definition.Binary); err == nil {
						toolIDs = append(toolIDs, definition.ID)
					}
				}
				sort.Strings(toolIDs)
				for _, id := range toolIDs {
					if _, err := fmt.Fprintln(cmd.OutOrStdout(), id); err != nil {
						return err
					}
				}
				return nil
			}

			var spinner *progressSpinner
			if !opts.JSON && !opts.NoProgress && showStatusSpinner() {
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
	cmd.Flags().BoolVar(&ids, "ids", false, "Print only installed tool IDs, one per line via PATH lookup (ignored with --json)")
	return cmd
}
