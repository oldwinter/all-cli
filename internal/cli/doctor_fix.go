package cli

import (
	"fmt"
	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
	"io"
	"strings"
	"time"
)

// Package installs need a longer timeout than status probes.
const doctorInstallTimeout = 10 * time.Minute

var doctorLookPath = execx.LookPath

func printDoctorFixes(w io.Writer, fixes model.DoctorFixRun) {
	fmt.Fprintln(w, "\nFixes")
	fmt.Fprintf(w, "  installer=%s dry_run=%t total=%d installed=%d preview=%d skipped=%d failed=%d\n",
		fixes.Installer,
		fixes.DryRun,
		fixes.Summary.Total,
		fixes.Summary.Installed,
		fixes.Summary.DryRun,
		fixes.Summary.Skipped,
		fixes.Summary.Failed,
	)
	if len(fixes.Items) == 0 {
		fmt.Fprintln(w, "  No missing tools to install.")
		return
	}
	for _, item := range fixes.Items {
		detail := item.Reason
		if len(item.Command) > 0 {
			detail = strings.Join(item.Command, " ")
			if item.Reason != "" {
				detail += " (" + item.Reason + ")"
			}
		}
		fmt.Fprintf(w, "  %-14s %-9s %s\n", item.ToolID, displayDoctorFixStatus(item.Status), detail)
	}
}

func displayDoctorFixStatus(status model.DoctorFixStatus) string {
	if status == model.DoctorFixDryRun {
		return "preview"
	}
	return string(status)
}

func doctorFixError(fixes model.DoctorFixRun) error {
	if fixes.Summary.Failed == 0 {
		return nil
	}
	return fmt.Errorf("%d install command(s) failed", fixes.Summary.Failed)
}
