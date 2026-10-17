package cli

import (
	"fmt"
	"strings"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/output"
	toolwrangler "github.com/oldwinter/all-cli/internal/tools/wrangler"
	"github.com/spf13/cobra"
)

func newWranglerCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wrangler",
		Short: "Inspect Wrangler login and account context",
	}

	cmd.AddCommand(newWranglerStatusCommand(opts, runner))
	cmd.AddCommand(newWranglerCurrentCommand(opts, runner))

	return cmd
}

func newWranglerStatusCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return newToolStatusCommand(opts, runner, "wrangler", "Show Wrangler status and current account context")
}

func newWranglerCurrentCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show Wrangler login state and account summary",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			a := toolwrangler.New(execx.TimeoutRunner{Runner: runner, Timeout: opts.Timeout})
			cur, warnings, errs, err := a.Current(ctx)
			if err != nil {
				errs = append(errs, err.Error())
			}

			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), newCurrentEnvelope(cur, warnings, errs))
			}

			printDiagnostics(cmd.ErrOrStderr(), warnings, errs)
			for _, key := range []string{"logged_in", "accounts_count", "account_id"} {
				if value := strings.TrimSpace(cur[key]); value != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", key, value)
				}
			}
			return nil
		},
	}
}
