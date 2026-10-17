package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/output"
	toolmise "github.com/oldwinter/all-cli/internal/tools/mise"
	"github.com/spf13/cobra"
)

func newMiseCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mise",
		Short: "Inspect resolved mise tool versions",
	}

	cmd.AddCommand(newMiseStatusCommand(opts, runner))
	cmd.AddCommand(newMiseCurrentCommand(opts, runner))

	return cmd
}

func newMiseStatusCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return newToolStatusCommand(opts, runner, "mise", "Show mise status and current resolved runtimes")
}

func newMiseCurrentCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show resolved mise runtime versions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			a := toolmise.New(execx.TimeoutRunner{Runner: runner, Timeout: opts.Timeout})
			cur, warnings, errs, err := a.Current(ctx)
			if err != nil {
				errs = append(errs, err.Error())
			}

			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), newCurrentEnvelope(cur, warnings, errs))
			}

			printDiagnostics(cmd.ErrOrStderr(), warnings, errs)
			keys := make([]string, 0, len(cur))
			for key := range cur {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				value := strings.TrimSpace(cur[key])
				fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", key, value)
			}
			return nil
		},
	}
}
