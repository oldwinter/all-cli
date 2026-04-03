package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func newSurpriseCommand() *cobra.Command {
	return &cobra.Command{
		Use:    "surprise",
		Hidden: true,
		Short:  "A small thank-you for curious users",
		Run: func(cmd *cobra.Command, _ []string) {
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, rainbowLine("    ★ all-cli ★"))
			fmt.Fprintln(out)
			fmt.Fprintln(out, strings.TrimSpace(`
  You read the flags. You ran the help. You typed a word
  that is not in the manual. That is the right energy.

  May your contexts always resolve, your tokens never leak,
  and your next deploy be boring in the best way.
`))
			fmt.Fprintln(out)
			fmt.Fprintln(out, dimIfTTY("  — the maintainers · 好奇的人运气不会太差"))
		},
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
