package cli

import (
	"fmt"
	"strings"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
	"github.com/oldwinter/all-cli/internal/output"
	"github.com/oldwinter/all-cli/internal/tools/glab"
	"github.com/spf13/cobra"
)

func newGLabCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "glab",
		Short: "Inspect/switch glab context (GitLab host)",
	}

	cmd.AddCommand(newGLabStatusCommand(opts, runner))
	cmd.AddCommand(newGLabCurrentCommand(opts, runner))
	cmd.AddCommand(newGLabListCommand(opts, runner))
	cmd.AddCommand(newGLabUseCommand(opts, runner))

	return cmd
}

func newGLabStatusCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show effective and global glab host context",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runGLabContextReport(cmd, opts, runner)
		},
	}
}

func newGLabCurrentCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show current glab context",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runGLabContextReport(cmd, opts, runner)
		},
	}
}

// runGLabContextReport is the shared body of `glab status` and `glab current`.
// Both commands stay independent values so neither mutates the other's metadata.
func runGLabContextReport(cmd *cobra.Command, opts *rootOptions, runner execx.Runner) error {
	ctx := cmd.Context()
	a := glab.New(execx.TimeoutRunner{Runner: runner, Timeout: opts.Timeout})
	cur, warnings, errs, _ := a.Current(ctx)

	if opts.JSON {
		return output.PrintJSON(cmd.OutOrStdout(), newCurrentEnvelope(cur, warnings, errs))
	}

	printDiagnostics(cmd.ErrOrStderr(), warnings, errs)

	if v := strings.TrimSpace(cur["effective_host"]); v != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "effective_host: %s\n", v)
	}
	if v := strings.TrimSpace(cur["global_host"]); v != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "global_host: %s\n", v)
	}
	if v := strings.TrimSpace(cur["user"]); v != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "user: %s\n", v)
	}
	return nil
}

func newGLabListCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all configured glab instances",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			a := glab.New(execx.TimeoutRunner{Runner: runner, Timeout: opts.Timeout})
			lst, warnings, errs, err := a.ListInstances(ctx)
			if err != nil {
				errs = append(errs, err.Error())
			}

			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), map[string]any{
					"instances": lst.Instances,
					"warnings":  nonNil(dedupeMessages(warnings)),
					"errors":    nonNil(dedupeMessages(errs)),
				})
			}

			printDiagnostics(cmd.ErrOrStderr(), warnings, errs)

			for _, inst := range lst.Instances {
				ok := "no"
				if inst.OK {
					ok = "yes"
				}
				user := ""
				if strings.TrimSpace(inst.User) != "" {
					user = " user=" + inst.User
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s ok=%s%s\n", inst.Host, ok, user)
			}
			return nil
		},
	}
}

func newGLabUseCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "use <host>",
		Short: "Set glab global default host",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			a := glab.New(execx.TimeoutRunner{Runner: runner, Timeout: opts.Timeout})
			host := args[0]
			after, err := a.UseHost(ctx, host)
			if err != nil {
				return reportUseFailure(cmd, opts, "glab", err)
			}
			cur, _, _, _ := a.Current(ctx)
			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), model.UseResult{OK: true, ToolID: "glab", Current: cur})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "set glab global host to %s\n", after)
			return nil
		},
	}
}
