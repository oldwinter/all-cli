package cli

import (
	"fmt"
	"strings"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/output"
	toolk9s "github.com/oldwinter/all-cli/internal/tools/k9s"
	"github.com/spf13/cobra"
)

func newK9sCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "k9s",
		Short: "Inspect k9s context and config path",
	}

	cmd.AddCommand(newK9sStatusCommand(opts, runner))
	cmd.AddCommand(newK9sCurrentCommand(opts, runner))

	return cmd
}

func newK9sStatusCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show k9s status and current context",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runSingleToolStatusCommand(cmd, opts, runner, "k9s")
		},
	}
}

func newK9sCurrentCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show current k9s context, namespace, and config path",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			a := toolk9s.New(execx.TimeoutRunner{Runner: runner, Timeout: opts.Timeout})
			cur, warnings, errs, _ := a.Current(ctx)

			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), map[string]any{
					"current":  cur,
					"warnings": dedupeMessages(warnings),
					"errors":   dedupeMessages(errs),
				})
			}

			printDiagnostics(cmd.ErrOrStderr(), warnings, errs)
			for _, key := range []string{"context", "namespace", "config"} {
				if value := strings.TrimSpace(cur[key]); value != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", key, value)
				}
			}
			return nil
		},
	}
}
