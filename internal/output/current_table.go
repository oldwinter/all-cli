package output

import (
	"fmt"
	"io"
	"strconv"
	"text/tabwriter"

	"github.com/oldwinter/all-cli/internal/model"
)

func PrintCurrentTable(w io.Writer, report model.StatusReport) {
	if len(report.Tools) == 0 {
		fmt.Fprintln(w, "No installed context-aware tools found.")
		fmt.Fprintln(w, "List tracked tools: all-cli catalog")
		fmt.Fprintln(w, "See install status: all-cli status")
		return
	}

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
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

func plainTableCell(value string) string {
	quoted := strconv.QuoteToGraphic(value)
	return quoted[1 : len(quoted)-1]
}
