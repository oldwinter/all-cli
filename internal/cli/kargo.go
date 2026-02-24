package cli

import (
	"fmt"
	"strings"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
	"github.com/oldwinter/all-cli/internal/output"
	"github.com/oldwinter/all-cli/internal/tools"
	"github.com/oldwinter/all-cli/internal/tools/kargo"
	"github.com/spf13/cobra"
)

func newKargoCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kargo",
		Short: "Inspect/switch Kargo CLI context",
	}

	cmd.AddCommand(newKargoStatusCommand(opts, runner))
	cmd.AddCommand(newKargoCurrentCommand(opts, runner))
	cmd.AddCommand(newKargoUseCommand(opts, runner))

	return cmd
}

func newKargoStatusCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show kargo status and current context",
		RunE: func(cmd *cobra.Command, _ []string) error {
			runnerT := execx.TimeoutRunner{Runner: runner, Timeout: opts.Timeout}
			def, ok := tools.FindByID("kargo")
			if !ok {
				return fmt.Errorf("kargo tool definition not found")
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

func newKargoCurrentCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show current kargo API/project context",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			a := kargo.New(execx.TimeoutRunner{Runner: runner, Timeout: opts.Timeout})
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
			if v := strings.TrimSpace(cur["api_address"]); v != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "api_address: %s\n", v)
			}
			if v := strings.TrimSpace(cur["project"]); v != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "project: %s\n", v)
			}
			return nil
		},
	}
}

func newKargoUseCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	var unset bool

	cmd := &cobra.Command{
		Use:   "use [project]",
		Short: "Set default kargo project (or unset it)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			a := kargo.New(execx.TimeoutRunner{Runner: runner, Timeout: opts.Timeout})

			if unset && len(args) > 0 {
				return fmt.Errorf("use either --unset or a project name, not both")
			}

			project := ""
			if unset {
				project = ""
			} else if len(args) == 1 {
				project = args[0]
			} else {
				return fmt.Errorf("project name is required (or use --unset)")
			}

			if err := a.SetDefaultProject(ctx, project); err != nil {
				if opts.JSON {
					_ = output.PrintJSON(cmd.OutOrStdout(), model.UseResult{OK: false, ToolID: "kargo", Error: err.Error()})
				}
				return err
			}

			cur, _, _, _ := a.Current(ctx)
			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), model.UseResult{OK: true, ToolID: "kargo", Current: cur})
			}
			if strings.TrimSpace(project) == "" {
				fmt.Fprintln(cmd.OutOrStdout(), "unset kargo default project")
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "set kargo default project to %s\n", project)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&unset, "unset", false, "Unset the default project")
	return cmd
}
