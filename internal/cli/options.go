package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newOptionsCommand(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "options",
		Short: "Show global CLI options inherited by subcommands",
		Long: `Prints the effective values of persistent flags (--json, --timeout) that apply
to all-cli and its subcommands. Similar to kubectl options.`,
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "json=%v\n", opts.JSON)
			fmt.Fprintf(cmd.OutOrStdout(), "timeout=%s\n", opts.Timeout)
		},
	}
}
