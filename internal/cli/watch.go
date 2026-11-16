package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
	"github.com/oldwinter/all-cli/internal/output"
	"github.com/oldwinter/all-cli/internal/tools"
	"github.com/spf13/cobra"
)

const (
	defaultWatchInterval = 2 * time.Second
	minWatchInterval     = 100 * time.Millisecond
)

type contextReportLoader func(context.Context) (model.StatusReport, error)

func newWatchCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	var toolsFilter string
	var categoriesFilter string
	var interval time.Duration
	var frames int
	var noClear bool

	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch current CLI contexts update live",
		Long: `Repeatedly refreshes the compact current-context view until interrupted.
Each refresh re-reads installed tools that expose context-like state and names
the tools whose context changed since the previous refresh.

Use --tools or --categories to limit external commands. Use --interval to choose
the refresh period, and --frames to stop after a fixed number of refreshes.
The screen is cleared between refreshes on an ANSI terminal; --no-clear keeps a
scrolling history instead. JSON is not available for this live view; use
'all-cli current --json' for machine-readable output.`,
		Example: `  all-cli watch
  all-cli watch --interval 1s --tools kubectl,docker
  all-cli watch --categories cloud --no-clear
  all-cli watch --frames 2 --interval 100ms --no-clear`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if opts.JSON {
				return fmt.Errorf("watch is a live text view; use 'all-cli current --json' for machine-readable output")
			}
			if interval < minWatchInterval {
				return fmt.Errorf("invalid --interval: must be at least %s, got %s", minWatchInterval, interval)
			}
			if frames < 0 {
				return fmt.Errorf("invalid --frames: must be zero or greater, got %d", frames)
			}

			loader, err := newContextReportLoader(runner, opts.timeouts(), toolsFilter, categoriesFilter)
			if err != nil {
				return err
			}
			clear := !noClear && terminalAnsiEnabled(os.Stdout)
			return runContextWatch(cmd.Context(), cmd.OutOrStdout(), interval, clear, frames, loader, time.Now)
		},
	}

	cmd.Flags().StringVar(&toolsFilter, "tools", "", "Comma-separated tool IDs to watch (e.g. kubectl,docker)")
	cmd.Flags().StringVar(&categoriesFilter, "categories", "", "Comma-separated categories to watch (e.g. cloud,k8s)")
	cmd.Flags().DurationVar(&interval, "interval", defaultWatchInterval, "Refresh interval (minimum 100ms)")
	cmd.Flags().IntVar(&frames, "frames", 0, "Stop after this many refreshes (0 means until interrupted)")
	cmd.Flags().BoolVar(&noClear, "no-clear", false, "Keep previous refreshes in the terminal scrollback")
	return cmd
}

func newContextReportLoader(
	runner execx.Runner,
	timeouts timeoutSetting,
	toolsFilter,
	categoriesFilter string,
) (contextReportLoader, error) {
	registry, err := registryForToolsFilter(toolsFilter)
	if err != nil {
		return nil, err
	}
	registry, err = registryForCategoriesFilter(registry, categoriesFilter)
	if err != nil {
		return nil, err
	}
	contextRegistry := make([]tools.ToolDefinition, 0, len(registry))
	for _, definition := range registry {
		if definition.Capabilities.HasContexts {
			contextRegistry = append(contextRegistry, definition)
		}
	}

	return func(ctx context.Context) (model.StatusReport, error) {
		report := evaluateStatusRegistry(ctx, contextRegistry, runner, timeouts, nil)
		installed := make([]model.ToolSummary, 0, len(report.Tools))
		for _, tool := range report.Tools {
			if tool.Installed {
				installed = append(installed, tool)
			}
		}
		report.Tools = installed
		sortToolSummaries(report.Tools, statusSortTool)
		return report, nil
	}, nil
}

func runContextWatch(
	ctx context.Context,
	w io.Writer,
	interval time.Duration,
	clear bool,
	frames int,
	load contextReportLoader,
	now func() time.Time,
) error {
	var previous map[string]map[string]string
	shown := 0

	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
		}

		report, err := load(ctx)
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return nil
		}

		current := contextValuesByTool(report.Tools)
		var changed []string
		if shown > 0 {
			changed = changedContextToolIDs(previous, current)
		}
		if shown > 0 && !clear {
			fmt.Fprintln(w)
		}
		output.PrintContextWatchFrame(w, report, output.ContextWatchOptions{
			At:          now(),
			Interval:    interval,
			Clear:       clear,
			Changed:     changed,
			ShowChanged: shown > 0,
		})
		previous = current
		shown++

		if frames > 0 && shown >= frames {
			return nil
		}
		timer.Reset(interval)
	}
}

func contextValuesByTool(toolsList []model.ToolSummary) map[string]map[string]string {
	valuesByTool := make(map[string]map[string]string, len(toolsList))
	for _, tool := range toolsList {
		values := make(map[string]string, len(tool.Current))
		for key, value := range tool.Current {
			value = strings.TrimSpace(value)
			if value != "" {
				values[key] = value
			}
		}
		valuesByTool[tool.ID] = values
	}
	return valuesByTool
}

func changedContextToolIDs(previous, current map[string]map[string]string) []string {
	changes := make([]string, 0)
	for id, values := range current {
		oldValues, existed := previous[id]
		switch {
		case !existed:
			changes = append(changes, id+" (new)")
		case !sameContextValues(oldValues, values):
			changes = append(changes, id)
		}
	}
	for id := range previous {
		if _, exists := current[id]; !exists {
			changes = append(changes, id+" (removed)")
		}
	}
	sort.Strings(changes)
	return changes
}

func sameContextValues(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}
