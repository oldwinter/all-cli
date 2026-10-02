package cli

import (
	"context"
	"strings"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
)

func buildReportStatus(ctx context.Context, runner execx.Runner, timeout time.Duration, toolsFilter, categoriesFilter string) (model.StatusReport, error) {
	reg, err := registryForToolsFilter(toolsFilter)
	if err != nil {
		return model.StatusReport{}, err
	}
	reg, err = registryForCategoriesFilter(reg, categoriesFilter)
	if err != nil {
		return model.StatusReport{}, err
	}
	report := evaluateStatusRegistry(ctx, reg, runner, timeout, nil)
	sortToolSummaries(report.Tools, statusSortTool)
	return report, nil
}

func selectedReportCategories(raw string) (map[string]bool, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	reg, err := registryForCategoriesFilter(defaultRegistry(), raw)
	if err != nil {
		return nil, err
	}
	selected := make(map[string]bool)
	for _, def := range reg {
		selected[strings.ToLower(def.Category)] = true
	}
	return selected, nil
}

func filterReportSnapshotCategories(summaries []model.ToolSummary, selected map[string]bool) []model.ToolSummary {
	if selected == nil {
		return summaries
	}
	filtered := make([]model.ToolSummary, 0, len(summaries))
	for _, tool := range summaries {
		if selected[strings.ToLower(tool.Category)] {
			filtered = append(filtered, tool)
		}
	}
	return filtered
}
