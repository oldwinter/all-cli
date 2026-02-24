package cli

import (
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/spf13/cobra"
)

var (
	// injected via -ldflags
	version = "dev"
	commit  = ""
	date    = ""
)

type rootOptions struct {
	JSON    bool
	Timeout time.Duration
}

func NewRootCommand() *cobra.Command {
	opts := &rootOptions{
		Timeout: 5 * time.Second,
	}
	runner := execx.DefaultRunner{}

	cmd := &cobra.Command{
		Use:   "all-cli",
		Short: "Inspect and manage common CLI tool contexts",
	}

	cmd.PersistentFlags().BoolVar(&opts.JSON, "json", false, "Output JSON")
	cmd.PersistentFlags().DurationVar(&opts.Timeout, "timeout", opts.Timeout, "External command timeout (e.g. 3s)")

	cmd.AddCommand(newStatusCommand(opts, runner))
	cmd.AddCommand(newVersionCommand())

	cmd.AddCommand(newKubectlCommand(opts, runner))
	cmd.AddCommand(newDockerCommand(opts, runner))
	cmd.AddCommand(newGHCommand(opts, runner))
	cmd.AddCommand(newGLabCommand(opts, runner))
	cmd.AddCommand(newArgoCDCommand(opts, runner))
	cmd.AddCommand(newKargoCommand(opts, runner))

	return cmd
}
