package output

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/oldwinter/all-cli/internal/model"
)

const (
	StatusTableGroupByNone     = "none"
	StatusTableGroupByCategory = "category"

	StatusTableSortTool         = "tool"
	StatusTableSortToolDesc     = "tool-desc"
	StatusTableSortCategory     = "category"
	StatusTableSortCategoryDesc = "category-desc"
)

type StatusTableOptions struct {
	GroupBy string
	SortBy  string
}

func PrintStatusTable(w io.Writer, report model.StatusReport) {
	PrintStatusTableWithOptions(w, report, StatusTableOptions{
		GroupBy: StatusTableGroupByNone,
		SortBy:  StatusTableSortTool,
	})
}

func PrintStatusTableWithOptions(w io.Writer, report model.StatusReport, opts StatusTableOptions) {
	groupBy := strings.ToLower(strings.TrimSpace(opts.GroupBy))
	switch groupBy {
	case StatusTableGroupByCategory:
		printStatusTableGroupedByCategory(w, report, opts)
	default:
		printStatusTableFlat(w, report)
	}
	printStatusDiagnostics(w, report)
}

func printStatusTableFlat(w io.Writer, report model.StatusReport) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "TOOL\tCATEGORY\tINSTALLED\tCONFIGURED\tCURRENT")

	for _, tool := range report.Tools {
		current := formatCurrentSummary(tool)
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			tool.ID,
			tool.Category,
			boolToYesNo(tool.Installed),
			string(tool.ConfiguredState),
			current,
		)
	}

	_ = tw.Flush()
}

func printStatusTableGroupedByCategory(w io.Writer, report model.StatusReport, opts StatusTableOptions) {
	groups := map[string][]model.ToolSummary{}
	seen := map[string]bool{}
	categories := make([]string, 0, len(report.Tools))

	for _, tool := range report.Tools {
		category := strings.TrimSpace(tool.Category)
		if category == "" {
			category = "uncategorized"
		}
		if !seen[category] {
			categories = append(categories, category)
			seen[category] = true
		}
		groups[category] = append(groups[category], tool)
	}

	sort.Strings(categories)
	if strings.EqualFold(strings.TrimSpace(opts.SortBy), StatusTableSortCategoryDesc) {
		for i, j := 0, len(categories)-1; i < j; i, j = i+1, j-1 {
			categories[i], categories[j] = categories[j], categories[i]
		}
	}

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "CATEGORY\tTOOL\tINSTALLED\tCONFIGURED\tCURRENT")

	for _, category := range categories {
		toolsInCategory := groups[category]
		for i, tool := range toolsInCategory {
			categoryLabel := ""
			if i == 0 {
				categoryLabel = category
			}
			current := formatCurrentSummary(tool)
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
				categoryLabel,
				tool.ID,
				boolToYesNo(tool.Installed),
				string(tool.ConfiguredState),
				current,
			)
		}
	}

	_ = tw.Flush()
}

func formatCurrentSummary(tool model.ToolSummary) string {
	if len(tool.Current) == 0 {
		return ""
	}

	keys := make([]string, 0, len(tool.Current))
	for k := range tool.Current {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		v := strings.TrimSpace(tool.Current[k])
		if v == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(parts, " ")
}

func boolToYesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func printStatusDiagnostics(w io.Writer, report model.StatusReport) {
	warnings := collectToolMessages(report.Tools, func(tool model.ToolSummary) []string {
		return tool.Warnings
	})
	errors := collectToolMessages(report.Tools, func(tool model.ToolSummary) []string {
		return tool.Errors
	})

	if len(warnings) == 0 && len(errors) == 0 {
		return
	}

	fmt.Fprintln(w)
	if len(warnings) > 0 {
		fmt.Fprintln(w, "Warnings:")
		for _, line := range warnings {
			fmt.Fprintf(w, "- %s\n", line)
		}
	}
	if len(errors) > 0 {
		if len(warnings) > 0 {
			fmt.Fprintln(w)
		}
		fmt.Fprintln(w, "Errors:")
		for _, line := range errors {
			fmt.Fprintf(w, "- %s\n", line)
		}
	}
}

func collectToolMessages(tools []model.ToolSummary, pick func(model.ToolSummary) []string) []string {
	var out []string
	for _, tool := range tools {
		for _, msg := range pick(tool) {
			msg = strings.TrimSpace(msg)
			if msg == "" {
				continue
			}
			out = append(out, fmt.Sprintf("%s: %s", tool.ID, msg))
		}
	}
	return out
}
