package output

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/oldwinter/all-cli/internal/model"
)

type ContextWatchOptions struct {
	At          time.Time
	Interval    time.Duration
	Clear       bool
	Changed     []string
	ShowChanged bool
}

func PrintContextWatchFrame(w io.Writer, report model.StatusReport, opts ContextWatchOptions) {
	if opts.Clear {
		fmt.Fprint(w, "\033[2J\033[H")
	}
	fmt.Fprintf(w, "all-cli watch  refresh=%s  at=%s\n\n", opts.Interval, opts.At.Format("15:04:05"))
	PrintCurrentTable(w, report)
	fmt.Fprintln(w)
	PrintCurrentSummary(w, report)
	if opts.ShowChanged {
		fmt.Fprintf(w, "Changed: %s\n", formatWatchChanges(opts.Changed))
	}
}

func formatWatchChanges(changes []string) string {
	if len(changes) == 0 {
		return "none"
	}
	return strings.Join(changes, ", ")
}
