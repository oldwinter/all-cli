package cli

import (
	"fmt"
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

func newStatusCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	var toolsFilter string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show installed/configured status for common CLI tools",
		RunE: func(cmd *cobra.Command, _ []string) error {
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

			report := model.StatusReport{
				SchemaVersion: model.SchemaVersionV01,
				GeneratedAt:   time.Now(),
				Tools:         make([]model.ToolSummary, len(reg)),
			}

			baseCtx := cmd.Context()
			var wg sync.WaitGroup
			for i, def := range reg {
				wg.Add(1)
				go func(i int, def tools.ToolDefinition) {
					defer wg.Done()
					report.Tools[i] = tools.Evaluate(baseCtx, def, runnerT)
				}(i, def)
			}
			wg.Wait()

			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), report)
			}
			output.PrintStatusTable(cmd.OutOrStdout(), report)
			return nil
		},
	}

	cmd.Flags().StringVar(&toolsFilter, "tools", "", "Comma-separated tool IDs to check (e.g. kubectl,docker)")
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
