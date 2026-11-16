package output

import (
	"fmt"
	"io"
	"time"

	"github.com/oldwinter/all-cli/internal/model"
)

// PrintCurrentMarkdown writes the filtered current-context view as a
// paste-ready Markdown table for issue, pull request, and incident notes.
func PrintCurrentMarkdown(w io.Writer, report model.StatusReport) {
	fmt.Fprintln(w, "# all-cli current contexts")
	if !report.GeneratedAt.IsZero() {
		fmt.Fprintf(w, "\nGenerated: `%s`\n", report.GeneratedAt.UTC().Format(time.RFC3339))
	}

	if len(report.Tools) == 0 {
		fmt.Fprintln(w, "\nNo context-aware tools in this view.")
		fmt.Fprintln(w, "\n- Show all current contexts: `all-cli current`")
		fmt.Fprintln(w, "- See install status: `all-cli status`")
		return
	}

	fmt.Fprintln(w, "\n| Tool | Current context |")
	fmt.Fprintln(w, "| --- | --- |")
	for _, tool := range report.Tools {
		current := formatCurrentSummary(tool)
		if current == "" {
			current = "none"
		}
		fmt.Fprintf(w, "| %s | %s |\n",
			markdownTableCell(tool.ID),
			markdownTableCell(current),
		)
	}

	total, readable := currentCounts(report)
	fmt.Fprintf(w, "\nCurrent: %d tools, %d with a readable context\n", total, readable)

	printMarkdownMessages(w, "Warnings", report.Tools, func(tool model.ToolSummary) []string {
		return tool.Warnings
	})
	printMarkdownMessages(w, "Errors", report.Tools, func(tool model.ToolSummary) []string {
		return tool.Errors
	})
}
