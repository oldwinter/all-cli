package output

import (
	"fmt"
	"io"
	"strconv"

	"github.com/oldwinter/all-cli/internal/model"
)

func PrintCurrentTable(w io.Writer, report model.StatusReport) {
	if len(report.Tools) == 0 {
		fmt.Fprintln(w, "No installed context-aware tools found.")
		fmt.Fprintln(w, "List tracked tools: all-cli catalog")
		fmt.Fprintln(w, "See install status: all-cli status")
		return
	}

	tw := NewTabWriter(w)
	fmt.Fprintln(tw, "TOOL\tCURRENT")
	for _, tool := range report.Tools {
		current := formatCurrentSummary(tool)
		if current == "" {
			current = "none"
		}
		fmt.Fprintf(tw, "%s\t%s\n", plainTableCell(tool.ID), plainTableCell(current))
	}
	_ = tw.Flush()

	printStatusDiagnostics(w, report)
}

// PrintUnresolvedCurrentTable keeps the empty result useful when a caller has
// explicitly asked for tools without a readable current context.
func PrintUnresolvedCurrentTable(w io.Writer, report model.StatusReport) {
	if len(report.Tools) == 0 {
		fmt.Fprintln(w, "No installed context-aware tools without a readable current context.")
		fmt.Fprintln(w, "Check all installed context tools: all-cli current")
		return
	}
	PrintCurrentTable(w, report)
}

// PrintCurrentSummary prints a compact count of installed context-aware tools
// and the ones for which all-cli could read at least one current-context value.
func PrintCurrentSummary(w io.Writer, report model.StatusReport) {
	total, active := currentCounts(report)
	fmt.Fprintf(w, "Current: %d tools, %d with a readable context\n", total, active)
}

func currentCounts(report model.StatusReport) (int, int) {
	active := 0
	for _, tool := range report.Tools {
		if formatCurrentSummary(tool) != "" {
			active++
		}
	}
	return len(report.Tools), active
}

func plainTableCell(value string) string {
	quoted := strconv.QuoteToGraphic(value)
	return quoted[1 : len(quoted)-1]
}
