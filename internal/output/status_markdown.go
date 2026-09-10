package output

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/oldwinter/all-cli/internal/model"
)

// PrintStatusMarkdown writes a paste-ready status report for issues and pull requests.
func PrintStatusMarkdown(w io.Writer, report model.StatusReport) {
	fmt.Fprintln(w, "# all-cli status report")
	if !report.GeneratedAt.IsZero() {
		fmt.Fprintf(w, "\nGenerated: `%s`\n", report.GeneratedAt.UTC().Format(time.RFC3339))
	}

	fmt.Fprintln(w, "\n| Tool | Category | Installed | Configured | Current |")
	fmt.Fprintln(w, "| --- | --- | --- | --- | --- |")
	for _, tool := range report.Tools {
		fmt.Fprintf(w, "| %s | %s | %s | %s | %s |\n",
			markdownTableCell(tool.ID),
			markdownTableCell(tool.Category),
			boolToYesNo(tool.Installed),
			markdownTableCell(string(tool.ConfiguredState)),
			markdownTableCell(formatCurrentSummary(tool)),
		)
	}

	printMarkdownMessages(w, "Warnings", report.Tools, func(tool model.ToolSummary) []string {
		return tool.Warnings
	})
	printMarkdownMessages(w, "Errors", report.Tools, func(tool model.ToolSummary) []string {
		return tool.Errors
	})
}

func printMarkdownMessages(
	w io.Writer,
	heading string,
	tools []model.ToolSummary,
	pick func(model.ToolSummary) []string,
) {
	lines := make([]string, 0)
	for _, tool := range tools {
		for _, message := range pick(tool) {
			message = strings.TrimSpace(message)
			if message == "" {
				continue
			}
			lines = append(lines, fmt.Sprintf("- `%s`: %s", tool.ID, markdownLine(message)))
		}
	}
	if len(lines) == 0 {
		return
	}

	fmt.Fprintf(w, "\n## %s\n\n", heading)
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
}

func markdownTableCell(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "|", `\|`)
	return markdownLine(value)
}

func markdownLine(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "<br>")
	value = strings.ReplaceAll(value, "\n", "<br>")
	return strings.ReplaceAll(value, "\r", "<br>")
}
