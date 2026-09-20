package output

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/oldwinter/all-cli/internal/model"
)

func PrintDiagnosticReport(w io.Writer, report model.DiagnosticReport) {
	fmt.Fprintf(w, "Diagnostics: total=%d info=%d warning=%d error=%d profile=%s\n",
		report.Summary.Total,
		report.Summary.Info,
		report.Summary.Warning,
		report.Summary.Error,
		report.Profile,
	)
	if len(report.Diagnostics) == 0 {
		fmt.Fprintln(w, "No diagnostics found.")
		return
	}
	for _, item := range report.Diagnostics {
		fmt.Fprintf(w, "\n[%s] %s: %s\n", item.Severity, item.RelatedTool, item.Problem)
		for _, evidence := range item.Evidence {
			fmt.Fprintf(w, "  evidence: %s\n", evidence)
		}
		for _, action := range item.SuggestedActions {
			fmt.Fprintf(w, "  action: %s", action.ID)
			if strings.TrimSpace(action.Title) != "" {
				fmt.Fprintf(w, " - %s", action.Title)
			}
			if len(action.Command) > 0 {
				fmt.Fprintf(w, " (%s)", strings.Join(action.Command, " "))
			}
			fmt.Fprintln(w)
		}
		fmt.Fprintf(w, "  safe_to_autofix: %t\n", item.SafeToAutofix)
	}
}

func PrintDoctorReport(w io.Writer, report model.DiagnosticReport) {
	fmt.Fprintln(w, "Doctor")
	PrintDiagnosticReport(w, report)
}

func PrintFixPlan(w io.Writer, plan model.FixPlan) {
	fmt.Fprintf(w, "Fix plan: dry_run=%t total=%d supported=%d blocked=%d\n",
		plan.DryRun,
		plan.Summary.Total,
		plan.Summary.Supported,
		plan.Summary.Blocked,
	)
	if len(plan.Items) == 0 {
		fmt.Fprintln(w, "No fixes planned.")
		return
	}
	for _, item := range plan.Items {
		fmt.Fprintf(w, "- %s %s supported=%t will_run=%t reason=%s\n",
			item.RelatedTool,
			item.Action.ID,
			item.Supported,
			item.WillRun,
			item.Reason,
		)
	}
}

func PrintSnapshotDiff(w io.Writer, report model.SnapshotDiffReport) {
	fmt.Fprintf(w, "Snapshot diff: added=%d removed=%d changed=%d\n",
		report.Summary.Added,
		report.Summary.Removed,
		report.Summary.Changed,
	)
	if len(report.Changes) == 0 {
		fmt.Fprintln(w, "No changes.")
		return
	}
	changes := append([]model.SnapshotToolChange(nil), report.Changes...)
	sort.SliceStable(changes, func(i, j int) bool {
		return changes[i].ToolID < changes[j].ToolID
	})
	for _, change := range changes {
		fields := ""
		if len(change.Fields) > 0 {
			fields = " fields=" + strings.Join(change.Fields, ",")
		}
		fmt.Fprintf(w, "- %s %s%s\n", change.ToolID, change.ChangeType, fields)
	}
}
