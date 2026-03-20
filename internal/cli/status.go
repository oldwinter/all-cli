package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

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
)

var (
	defaultRegistry   = tools.DefaultRegistry
	showStatusSpinner = func() bool {
		return isTerminal(os.Stderr)
	}
)

func newStatusCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	var toolsFilter string
	var groupBy string
	var sortBy string
	var quiet bool
	var installedOnly bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show installed/configured status for common CLI tools",
		RunE: func(cmd *cobra.Command, _ []string) error {
			groupByValue, err := parseStatusGroupBy(groupBy)
			if err != nil {
				return err
			}
			sortByValue, err := parseStatusSort(sortBy)
			if err != nil {
				return err
			}

			reg := defaultRegistry()
			if strings.TrimSpace(toolsFilter) != "" {
				filterSet, err := parseToolsFilter(toolsFilter)
				if err != nil {
					return err
				}
				var filtered []tools.ToolDefinition
				for _, def := range reg {
					if filterSet[def.ID] {
						filtered = append(filtered, def)
					}
				}
				reg = filtered
			}

			report := model.NewStatusReport(len(reg))
			report.GeneratedAt = time.Now()

			var spinner *progressSpinner
			if !opts.JSON && showStatusSpinner() {
				spinner = newProgressSpinner(cmd.ErrOrStderr(), len(reg))
				spinner.Start()
			}

			baseCtx := cmd.Context()
			var wg sync.WaitGroup
			for i, def := range reg {
				wg.Add(1)
				go func(i int, def tools.ToolDefinition) {
					defer wg.Done()
					report.Tools[i] = evaluateToolSummary(baseCtx, def, runnerForTool(runner, opts.Timeout, def))
					if spinner != nil {
						spinner.Inc(def.ID)
					}
				}(i, def)
			}
			wg.Wait()

			if spinner != nil {
				spinner.Stop()
			}

			sortToolSummaries(report.Tools, sortByValue)

			if installedOnly {
				filtered := report.Tools[:0]
				for _, t := range report.Tools {
					if t.Installed {
						filtered = append(filtered, t)
					}
				}
				report.Tools = filtered
			}
			if quiet {
				filtered := report.Tools[:0]
				for _, t := range report.Tools {
					if !t.Installed || t.ConfiguredState == model.ConfiguredNo ||
						len(t.Warnings) > 0 || len(t.Errors) > 0 {
						filtered = append(filtered, t)
					}
				}
				report.Tools = filtered
			}

			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), report)
			}
			output.PrintStatusTableWithOptions(cmd.OutOrStdout(), report, output.StatusTableOptions{
				GroupBy: groupByValue,
				SortBy:  sortByValue,
			})
			return nil
		},
	}

	cmd.Flags().StringVar(&toolsFilter, "tools", "", "Comma-separated tool IDs to check (e.g. kubectl,docker)")
	cmd.Flags().StringVar(&groupBy, "group-by", statusGroupByCategory, "Group output: category|none")
	cmd.Flags().StringVar(&sortBy, "sort", statusSortTool, "Sort order: tool|tool-desc|category|category-desc")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "Only show tools with issues (not installed, unconfigured, warnings, or errors)")
	cmd.Flags().BoolVar(&installedOnly, "installed-only", false, "Only show installed tools")
	return cmd
}

func runnerForTool(baseRunner execx.Runner, defaultTimeout time.Duration, def tools.ToolDefinition) execx.Runner {
	timeout := defaultTimeout
	if def.Timeout > 0 {
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
		sort.Strings(unknown)
		return nil, fmt.Errorf("unknown tool IDs: %s", strings.Join(unknown, ", "))
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
	case statusSortToolDesc, statusSortCategory, statusSortCategoryDesc:
		return value, nil
	default:
		return "", fmt.Errorf("invalid --sort value %q (allowed: tool, tool-desc, category, category-desc)", s)
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
