package output

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/oldwinter/all-cli/internal/model"
)

func PrintStatusTable(w io.Writer, report model.StatusReport) {
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
