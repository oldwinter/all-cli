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

func newStatusCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	var toolsFilter string
	var groupBy string
	var sortBy string

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

			runnerT := execx.TimeoutRunner{Runner: runner, Timeout: opts.Timeout}
			reg := tools.DefaultRegistry()
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
				if len(filtered) == 0 {
					return fmt.Errorf("no tools matched --tools=%q", toolsFilter)
				}
				reg = filtered
			}

			report := model.NewStatusReport(len(reg))
			report.GeneratedAt = time.Now()

			var spinner *progressSpinner
			if !opts.JSON && isTerminal(os.Stderr) {
				spinner = newProgressSpinner(cmd.ErrOrStderr(), len(reg))
				spinner.Start()
			}

			baseCtx := cmd.Context()
			var wg sync.WaitGroup
			for i, def := range reg {
				wg.Add(1)
				go func(i int, def tools.ToolDefinition) {
					defer wg.Done()
					report.Tools[i] = tools.Evaluate(baseCtx, def, runnerT)
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
	return cmd
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

	// Validate IDs
	valid := map[string]bool{}
	for _, def := range tools.DefaultRegistry() {
		valid[def.ID] = true
	}
	var unknown []string
	for id := range out {
		if !valid[id] {
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
