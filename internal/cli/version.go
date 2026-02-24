package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, _ []string) {
			v := version
			if commit != "" || date != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "%s (commit=%s date=%s)\n", v, commit, date)
				return
			}
			fmt.Fprintln(cmd.OutOrStdout(), v)
		},
	}
}
