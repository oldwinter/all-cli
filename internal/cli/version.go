package cli

import (
	"fmt"

	"github.com/oldwinter/all-cli/internal/output"
	"github.com/spf13/cobra"
)

type versionReport struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

func newVersionCommand(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(cmd *cobra.Command, _ []string) error {
			report := resolvedVersionReport()
			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), report)
			}
			fmt.Fprintln(cmd.OutOrStdout(), formatVersionReport(report))
			return nil
		},
	}
}
