package cli

import (
	"fmt"

	"github.com/oldwinter/all-cli/internal/output"
	"github.com/spf13/cobra"
)

type optionsReport struct {
	JSON       bool   `json:"json"`
	NoProgress bool   `json:"no_progress"`
	Timeout    string `json:"timeout"`
}

func newOptionsCommand(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "options",
		Short: "Show global CLI options inherited by subcommands",
		Long: `Prints the effective values of persistent flags (--json, --no-progress,
--timeout) that apply to all-cli and its subcommands. Similar to kubectl options.
Use --json for a machine-readable object.`,
		Example: `  all-cli options
  all-cli options --json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			noProgress := opts.NoProgress || allCliNoProgressEnvSet()
			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), optionsReport{
					JSON:       opts.JSON,
					NoProgress: noProgress,
					Timeout:    opts.Timeout.String(),
				})
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "json=%v\n", opts.JSON); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "no_progress=%v\n", noProgress); err != nil {
				return err
			}
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "timeout=%s\n", opts.Timeout)
			return err
		},
	}
}
