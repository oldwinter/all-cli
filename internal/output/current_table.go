package output

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/oldwinter/all-cli/internal/model"
)

func PrintCurrentTable(w io.Writer, report model.StatusReport) {
	if len(report.Tools) == 0 {
		fmt.Fprintln(w, "No installed context-aware tools found.")
		return
	}

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "TOOL\tCURRENT")
	for _, tool := range report.Tools {
		current := formatCurrentSummary(tool)
		if current == "" {
			current = "none"
		}
		fmt.Fprintf(tw, "%s\t%s\n", tool.ID, current)
	}
	_ = tw.Flush()

	printStatusDiagnostics(w, report)
}
