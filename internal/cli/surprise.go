package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/oldwinter/all-cli/internal/tools"
	"github.com/spf13/cobra"
)

type surpriseRecommendation struct {
	ToolID   string
	Category string
	Purpose  string
}

func newSurpriseCommand() *cobra.Command {
	return &cobra.Command{
		Use:    "surprise",
		Hidden: true,
		Short:  "A small thank-you for curious users",
		Run: func(cmd *cobra.Command, _ []string) {
			out := cmd.OutOrStdout()
			recommendation := dailySurpriseRecommendation(time.Now())
			fmt.Fprintln(out, rainbowLine("    ★ all-cli ★"))
			fmt.Fprintln(out)
			fmt.Fprintln(out, strings.TrimSpace(`
  You read the flags. You ran the help. You typed a word
  that is not in the manual. That is the right energy.

  May your contexts always resolve, your tokens never leak,
  and your next deploy be boring in the best way.
`))
			fmt.Fprintln(out)
			fmt.Fprintf(out, "  Tool of the day: %s (%s)\n", recommendation.ToolID, recommendation.Category)
			fmt.Fprintf(out, "  %s\n", recommendation.Purpose)
			fmt.Fprintf(out, "  Explore it: all-cli describe %s\n", recommendation.ToolID)
			fmt.Fprintln(out)
			fmt.Fprintln(out, dimIfTTY("  — the maintainers · 好奇的人运气不会太差"))
		},
	}
}

func dailySurpriseRecommendation(day time.Time) surpriseRecommendation {
	registry := tools.DefaultRegistry()
	sort.Slice(registry, func(i, j int) bool {
		return registry[i].ID < registry[j].ID
	})

	dayNumber := day.Year()*366 + day.YearDay()
	definition := registry[dayNumber%len(registry)]
	return surpriseRecommendation{
		ToolID:   definition.ID,
		Category: definition.Category,
		Purpose:  tools.MetadataForTool(definition.ID).Purpose,
	}
}

func rainbowLine(s string) string {
	if !terminalAnsiEnabled(os.Stdout) {
		return s
	}
	const reset = "\033[0m"
	attrs := []string{"\033[38;5;204m", "\033[38;5;209m", "\033[38;5;214m", "\033[38;5;220m", "\033[38;5;154m", "\033[38;5;80m", "\033[38;5;45m", "\033[38;5;63m"}
	var b strings.Builder
	i := 0
	for _, r := range s {
		if r == ' ' {
			b.WriteRune(r)
			continue
		}
		b.WriteString(attrs[i%len(attrs)])
		b.WriteRune(r)
		b.WriteString(reset)
		i++
	}
	return b.String()
}

func dimIfTTY(s string) string {
	if !terminalAnsiEnabled(os.Stdout) {
		return s
	}
	return "\033[2m" + s + "\033[0m"
}
