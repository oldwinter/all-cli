package cli

import (
	"fmt"
	"strings"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
	"github.com/oldwinter/all-cli/internal/output"
	"github.com/oldwinter/all-cli/internal/tools"
	"github.com/oldwinter/all-cli/internal/tools/docker"
	"github.com/spf13/cobra"
)

func newDockerCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docker",
		Short: "Manage docker contexts",
	}

	cmd.AddCommand(newDockerStatusCommand(opts, runner))
	cmd.AddCommand(newDockerCurrentCommand(opts, runner))
	cmd.AddCommand(newDockerListCommand(opts, runner))
	cmd.AddCommand(newDockerUseCommand(opts, runner))

	return cmd
}

func newDockerStatusCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show docker status and current context",
		RunE: func(cmd *cobra.Command, _ []string) error {
			runnerT := execx.TimeoutRunner{Runner: runner, Timeout: opts.Timeout}
			def, ok := tools.FindByID("docker")
			if !ok {
				return fmt.Errorf("docker tool definition not found")
			}
			summary := tools.Evaluate(cmd.Context(), def, runnerT)
			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), summary)
			}
			report := model.NewStatusReport(1)
			report.Tools[0] = summary
			output.PrintStatusTable(cmd.OutOrStdout(), report)
			return nil
		},
	}
}

func newDockerCurrentCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show current docker context",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			a := docker.New(execx.TimeoutRunner{Runner: runner, Timeout: opts.Timeout})
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
			if v := strings.TrimSpace(cur["context"]); v != "" {
				fmt.Fprintln(cmd.OutOrStdout(), v)
			}
			return nil
		},
	}
}

func newDockerListCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List docker contexts",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			a := docker.New(execx.TimeoutRunner{Runner: runner, Timeout: opts.Timeout})

			contexts, warnings, errs, err := a.ListContexts(ctx)
			if err != nil {
				errs = append(errs, err.Error())
			}
			cur, _, _, _ := a.Current(ctx)
			currentName := strings.TrimSpace(cur["context"])

			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), map[string]any{
					"current":  cur,
					"contexts": contexts,
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
			for _, c := range contexts {
				mark := " "
				if c.Name == currentName || c.IsCurrent {
					mark = "*"
				}
				desc := strings.TrimSpace(c.Description)
				if desc != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "%s %s\t%s\n", mark, c.Name, desc)
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", mark, c.Name)
				}
			}
			return nil
		},
	}
}

func newDockerUseCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "use <context>",
		Short: "Switch current docker context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			a := docker.New(execx.TimeoutRunner{Runner: runner, Timeout: opts.Timeout})

			contextName := args[0]
			if err := a.UseContext(ctx, contextName); err != nil {
				if opts.JSON {
					_ = output.PrintJSON(cmd.OutOrStdout(), model.UseResult{OK: false, ToolID: "docker", Error: err.Error()})
				}
				return err
			}

			cur, _, _, _ := a.Current(ctx)
			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), model.UseResult{OK: true, ToolID: "docker", Current: cur})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "switched docker context to %s\n", contextName)
			return nil
		},
	}
}
