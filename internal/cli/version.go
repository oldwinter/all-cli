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
			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), versionReport{
					Version: version,
					Commit:  commit,
					Date:    date,
				})
			}
			fmt.Fprintln(cmd.OutOrStdout(), VersionString())
			return nil
		},
	}
}
