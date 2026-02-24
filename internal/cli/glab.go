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
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			a := glab.New(execx.TimeoutRunner{Runner: runner, Timeout: opts.Timeout})
			cur, warnings, errs, err := a.Current(ctx)
			if err != nil {
				errs = append(errs, err.Error())
			}

			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), map[string]any{
					"current":  cur,
					"warnings": warnings,
					"errors":   errs,
				})
			}

			for _, w := range warnings {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", w)
			}
			for _, e := range errs {
				fmt.Fprintf(cmd.ErrOrStderr(), "error: %s\n", e)
			}

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
		},
	}
}

func newGLabCurrentCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	cmd := newGLabStatusCommand(opts, runner)
	cmd.Use = "current"
	cmd.Short = "Show current glab context"
	return cmd
}

func newGLabListCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all configured glab instances",
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
					"warnings":  warnings,
					"errors":    errs,
				})
			}

			for _, w := range warnings {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", w)
			}
			for _, e := range errs {
				fmt.Fprintf(cmd.ErrOrStderr(), "error: %s\n", e)
			}

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
				if opts.JSON {
					_ = output.PrintJSON(cmd.OutOrStdout(), model.UseResult{OK: false, ToolID: "glab", Error: err.Error()})
				}
				return err
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
