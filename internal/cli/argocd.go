package cli

import (
	"fmt"
	"strings"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
	"github.com/oldwinter/all-cli/internal/output"
	"github.com/oldwinter/all-cli/internal/tools"
	"github.com/oldwinter/all-cli/internal/tools/argocd"
	"github.com/spf13/cobra"
)

func newArgoCDCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "argocd",
		Short: "Inspect/switch Argo CD contexts",
	}

	cmd.AddCommand(newArgoCDStatusCommand(opts, runner))
	cmd.AddCommand(newArgoCDCurrentCommand(opts, runner))
	cmd.AddCommand(newArgoCDListCommand(opts, runner))
	cmd.AddCommand(newArgoCDUseCommand(opts, runner))

	return cmd
}

func newArgoCDStatusCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show argocd status and current context",
		RunE: func(cmd *cobra.Command, _ []string) error {
			runnerT := execx.TimeoutRunner{Runner: runner, Timeout: opts.Timeout}
			def, ok := tools.FindByID("argocd")
			if !ok {
				return fmt.Errorf("argocd tool definition not found")
			}
			summary := tools.Evaluate(cmd.Context(), def, runnerT)
			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), summary)
			}
			report := model.StatusReport{SchemaVersion: model.SchemaVersionV01, Tools: []model.ToolSummary{summary}}
			output.PrintStatusTable(cmd.OutOrStdout(), report)
			return nil
		},
	}
}

func newArgoCDCurrentCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show current argocd context",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			a := argocd.New(execx.TimeoutRunner{Runner: runner, Timeout: opts.Timeout})
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
				fmt.Fprintf(cmd.OutOrStdout(), "context: %s\n", v)
			}
			if v := strings.TrimSpace(cur["server"]); v != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "server: %s\n", v)
			}
			return nil
		},
	}
}

func newArgoCDListCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List argocd contexts",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			a := argocd.New(execx.TimeoutRunner{Runner: runner, Timeout: opts.Timeout})

			contexts, warnings, errs, err := a.ListContexts(ctx)
			if err != nil {
				errs = append(errs, err.Error())
			}

			var current map[string]string
			for _, c := range contexts {
				if c.IsCurrent {
					current = map[string]string{"context": c.Name}
					if strings.TrimSpace(c.Server) != "" {
						current["server"] = c.Server
					}
					break
				}
			}

			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), map[string]any{
					"current":  current,
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
				if c.IsCurrent {
					mark = "*"
				}
				server := strings.TrimSpace(c.Server)
				if server != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "%s %s (server=%s)\n", mark, c.Name, server)
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", mark, c.Name)
				}
			}
			return nil
		},
	}
}

func newArgoCDUseCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "use <context>",
		Short: "Switch current argocd context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			a := argocd.New(execx.TimeoutRunner{Runner: runner, Timeout: opts.Timeout})

			contextName := args[0]
			if err := a.UseContext(ctx, contextName); err != nil {
				if opts.JSON {
					_ = output.PrintJSON(cmd.OutOrStdout(), model.UseResult{OK: false, ToolID: "argocd", Error: err.Error()})
				}
				return err
			}

			cur, _, _, _ := a.Current(ctx)
			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), model.UseResult{OK: true, ToolID: "argocd", Current: cur})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "switched argocd context to %s\n", contextName)
			return nil
		},
	}
}
