package cli

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	diag "github.com/oldwinter/all-cli/internal/diagnose"
	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
	"github.com/oldwinter/all-cli/internal/output"
	"github.com/oldwinter/all-cli/internal/tools"
	"github.com/spf13/cobra"
)

const (
	statusGroupByNone     = "none"
	statusGroupByCategory = "category"

	statusSortTool         = "tool"
	statusSortToolDesc     = "tool-desc"
	statusSortCategory     = "category"
	statusSortCategoryDesc = "category-desc"
	statusSortAttention    = "attention"
)

var (
	defaultRegistry   = tools.DefaultRegistry
	showStatusSpinner = func() bool {
		return statusSpinnerEnabled()
	}
)

func newStatusCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	var toolsFilter string
	var categoriesFilter string
	var groupBy string
	var sortBy string
	var ids bool
	var summaryOnly bool
	var quiet bool
	var installedOnly bool
	var missingOnly bool
	var unconfiguredOnly bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show installed/configured status for common CLI tools",
		Long: `Evaluates each tracked tool in the built-in registry: installation path,
configuration state, capabilities, and optional current context snapshot.

The progress indicator may be shown on stderr while tools are checked, unless
--json, --ids, --summary-only, --quiet, or --no-progress is used.
Use --ids to print one checked tool ID per line. --json takes precedence over --ids.
Use --summary-only to print just the one-line summary count without tables or warning details.`,
		Example: `  all-cli status
  all-cli status --tools kubectl,docker --group-by none
  all-cli status --categories ai,cloud
  all-cli status --categories ai --missing-only
  all-cli status --missing-only --ids
  all-cli status --sort attention
  all-cli status --unconfigured-only --ids
  all-cli status --summary-only
  all-cli status --installed-only --quiet`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			groupByValue, err := parseStatusGroupBy(groupBy)
			if err != nil {
				return err
			}
			sortByValue, err := parseStatusSort(sortBy)
			if err != nil {
				return err
			}

			reg, err := registryForToolsFilter(toolsFilter)
			if err != nil {
				return err
			}
			reg, err = registryForCategoriesFilter(reg, categoriesFilter)
			if err != nil {
				return err
			}

			var spinner *progressSpinner
			if !opts.JSON && !ids && !summaryOnly && !opts.NoProgress && !quiet && showStatusSpinner() {
				spinner = newProgressSpinner(cmd.ErrOrStderr(), len(reg))
				spinner.Start()
			}

			report := evaluateStatusRegistry(cmd.Context(), reg, runner, opts.timeouts(), spinner)

			if spinner != nil {
				spinner.Stop()
			}

			sortToolSummaries(report.Tools, sortByValue)

			// Diagnostics describe the full evaluation, not the rendered subset.
			// Generating them after filtering --installed-only / --missing-only /
			// --quiet / --unconfigured-only silently dropped almost every
			// diagnostic: the filtered-out tools are exactly the ones that carry
			// "not installed" and "unconfigured" findings, so a caller asking
			// "what is wrong with what I have installed" received a near-empty,
			// falsely healthy report.
			var diagnostics []model.DiagnosticItem
			if opts.JSON {
				diagnostics = diag.Generate(report, diag.Options{Profile: diag.ProfileAgent}).Diagnostics
			}

			report.Tools = filterStatusTools(report.Tools, installedOnly, missingOnly, quiet, unconfiguredOnly)

			if opts.JSON {
				report.Diagnostics = diagnostics
				return output.PrintJSON(cmd.OutOrStdout(), report)
			}
			if ids {
				return printStatusIDs(cmd, report.Tools)
			}
			if summaryOnly {
				output.PrintStatusSummary(cmd.OutOrStdout(), report)
				return nil
			}
			output.PrintStatusTableWithOptions(cmd.OutOrStdout(), report, output.StatusTableOptions{
				GroupBy: groupByValue,
				SortBy:  sortByValue,
			})
			return nil
		},
	}

	cmd.Flags().StringVar(&toolsFilter, "tools", "", "Comma-separated tool IDs to check (e.g. kubectl,docker)")
	cmd.Flags().StringVar(&categoriesFilter, "categories", "", "Comma-separated categories to check (e.g. ai,cloud)")
	cmd.Flags().StringVar(&groupBy, "group-by", statusGroupByCategory, "Group output: category|none")
	cmd.Flags().StringVar(&sortBy, "sort", statusSortTool, "Sort order: tool|tool-desc|category|category-desc|attention")
	cmd.Flags().BoolVar(&ids, "ids", false, "Print only checked tool IDs, one per line (ignored with --json)")
	cmd.Flags().BoolVar(&summaryOnly, "summary-only", false, "Print only the one-line summary count (ignored with --json)")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "Only show tools with issues (not installed, unconfigured, warnings, or errors)")
	cmd.Flags().BoolVar(&installedOnly, "installed-only", false, "Only show installed tools")
	cmd.Flags().BoolVar(&missingOnly, "missing-only", false, "Only show tools that are not installed")
	cmd.Flags().BoolVar(&unconfiguredOnly, "unconfigured-only", false, "Only show installed tools that are not configured")
	cmd.MarkFlagsMutuallyExclusive("installed-only", "missing-only")
	cmd.MarkFlagsMutuallyExclusive("unconfigured-only", "missing-only")
	cmd.MarkFlagsMutuallyExclusive("ids", "summary-only")
	return cmd
}

func printStatusIDs(cmd *cobra.Command, toolsList []model.ToolSummary) error {
	for _, tool := range toolsList {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), tool.ID); err != nil {
			return err
		}
	}
	return nil
}

func filterStatusTools(toolsList []model.ToolSummary, installedOnly, missingOnly, quiet, unconfiguredOnly bool) []model.ToolSummary {
	// Allocate rather than reusing toolsList[:0]: the function reads as a pure
	// filter, and overwriting the caller's backing array would corrupt any
	// caller that keeps the original slice.
	filtered := make([]model.ToolSummary, 0, len(toolsList))
	for _, tool := range toolsList {
		if installedOnly && !tool.Installed {
			continue
		}
		if missingOnly && tool.Installed {
			continue
		}
		if unconfiguredOnly && (!tool.Installed || tool.ConfiguredState != model.ConfiguredNo) {
			continue
		}
		if quiet && !toolNeedsAttention(tool) {
			continue
		}
		filtered = append(filtered, tool)
	}
	return filtered
}

// toolNeedsAttention mirrors the --quiet filter: anything not installed,
// unconfigured, or carrying warnings/errors.
func toolNeedsAttention(tool model.ToolSummary) bool {
	if !tool.Installed {
		return true
	}
	if tool.ConfiguredState == model.ConfiguredNo {
		return true
	}
	return len(tool.Warnings) > 0 || len(tool.Errors) > 0
}

// buildStatusReportWithTimeoutSettings evaluates the registry with the user's
// full --timeout intent, so an explicit flag stays authoritative for tools that
// declare their own default timeout.
func buildStatusReportWithTimeoutSettings(ctx context.Context, runner execx.Runner, timeouts timeoutSetting, toolsFilter string) (model.StatusReport, error) {
	reg, err := registryForToolsFilter(toolsFilter)
	if err != nil {
		return model.StatusReport{}, err
	}
	report := evaluateStatusRegistry(ctx, reg, runner, timeouts, nil)
	sortToolSummaries(report.Tools, statusSortTool)
	return report, nil
}

// buildStatusReport is the Duration-only form of
// buildStatusReportWithTimeoutSettings. A bare Duration carries no knowledge of
// whether the user passed --timeout, so it is treated as a default: tool
// definitions may still substitute their own value.
func buildStatusReport(ctx context.Context, runner execx.Runner, defaultTimeout time.Duration, toolsFilter string) (model.StatusReport, error) {
	return buildStatusReportWithTimeoutSettings(ctx, runner, timeoutSetting{Value: defaultTimeout}, toolsFilter)
}

func registryForToolsFilter(toolsFilter string) ([]tools.ToolDefinition, error) {
	reg := defaultRegistry()
	if strings.TrimSpace(toolsFilter) == "" {
		return reg, nil
	}
	filterSet, err := parseToolsFilter(toolsFilter)
	if err != nil {
		return nil, err
	}
	filtered := make([]tools.ToolDefinition, 0, len(filterSet))
	for _, def := range reg {
		if filterSet[def.ID] {
			filtered = append(filtered, def)
		}
	}
	return filtered, nil
}

func evaluateStatusRegistry(ctx context.Context, reg []tools.ToolDefinition, runner execx.Runner, timeouts timeoutSetting, spinner *progressSpinner) model.StatusReport {
	report := model.NewStatusReport(len(reg))
	report.GeneratedAt = time.Now()

	var wg sync.WaitGroup
	for i, def := range reg {
		wg.Add(1)
		go func(i int, def tools.ToolDefinition) {
			defer wg.Done()
			report.Tools[i] = evaluateToolSummary(ctx, def, runnerForTool(runner, timeouts, def))
			if spinner != nil {
				spinner.Inc(def.ID)
			}
		}(i, def)
	}
	wg.Wait()
	return report
}

// runnerForTool builds the runner used to inspect one tool.
//
// Precedence: an explicit --timeout always wins. A tool definition's Timeout is
// a fallback for tools whose own commands are legitimately slow, not an
// override of the user's decision: silently replacing `--timeout 100ms` with
// 10s made the flag a lie for wrangler/vercel/railway/netlify.
func runnerForTool(baseRunner execx.Runner, timeouts timeoutSetting, def tools.ToolDefinition) execx.Runner {
	timeout := timeouts.Value
	if !timeouts.Explicit && def.Timeout > 0 {
		timeout = def.Timeout
	}
	return execx.TimeoutRunner{Runner: baseRunner, Timeout: timeout}
}

func parseToolsFilter(s string) (map[string]bool, error) {
	out := map[string]bool{}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out[part] = true
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("invalid --tools value")
	}

	var unknown []string
	for id := range out {
		if _, ok := tools.FindByID(id); !ok {
			unknown = append(unknown, id)
		}
	}
	if len(unknown) > 0 {
		return nil, unknownToolIDError(unknown)
	}

	return out, nil
}

func parseStatusGroupBy(s string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(s))
	switch value {
	case "", statusGroupByCategory:
		return statusGroupByCategory, nil
	case statusGroupByNone:
		return statusGroupByNone, nil
	default:
		return "", fmt.Errorf("invalid --group-by value %q (allowed: category, none)", s)
	}
}

func parseStatusSort(s string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(s))
	switch value {
	case "", statusSortTool:
		return statusSortTool, nil
	case statusSortToolDesc, statusSortCategory, statusSortCategoryDesc, statusSortAttention:
		return value, nil
	default:
		return "", fmt.Errorf("invalid --sort value %q (allowed: tool, tool-desc, category, category-desc, attention)", s)
	}
}

func sortToolSummaries(toolsList []model.ToolSummary, sortBy string) {
	less := func(i, j int) bool {
		left := toolsList[i]
		right := toolsList[j]

		switch sortBy {
		case statusSortToolDesc:
			if left.ID != right.ID {
				return left.ID > right.ID
			}
			return left.Category < right.Category
		case statusSortCategory:
			if left.Category != right.Category {
				return left.Category < right.Category
			}
			return left.ID < right.ID
		case statusSortCategoryDesc:
			if left.Category != right.Category {
				return left.Category > right.Category
			}
			return left.ID < right.ID
		case statusSortAttention:
			leftAttention := toolNeedsAttention(left)
			rightAttention := toolNeedsAttention(right)
			if leftAttention != rightAttention {
				return leftAttention && !rightAttention
			}
			if left.ID != right.ID {
				return left.ID < right.ID
			}
			return left.Category < right.Category
		case statusSortTool:
			fallthrough
		default:
			if left.ID != right.ID {
				return left.ID < right.ID
			}
			return left.Category < right.Category
		}
	}

	sort.SliceStable(toolsList, less)
}
