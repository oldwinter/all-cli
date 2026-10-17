package cli

import (
	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/spf13/cobra"
)

// newToolStatusCommand builds the per-tool `status` subcommand shared by every
// tool integration. Each tool differs only in its ID and one-line summary, so a
// single builder keeps the nine call sites identical and guarantees consistent
// argument validation.
func newToolStatusCommand(opts *rootOptions, runner execx.Runner, toolID, short string) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runSingleToolStatusCommand(cmd, opts, runner, toolID)
		},
	}
}
