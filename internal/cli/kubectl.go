package cli

import (
	"fmt"
	"strings"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
	"github.com/oldwinter/all-cli/internal/output"
	"github.com/oldwinter/all-cli/internal/tools"
	"github.com/oldwinter/all-cli/internal/tools/kubectl"
	"github.com/spf13/cobra"
)

func newKubectlCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kubectl",
		Short: "Manage kubectl contexts and namespace",
	}

	cmd.AddCommand(newKubectlStatusCommand(opts, runner))
	cmd.AddCommand(newKubectlCurrentCommand(opts, runner))
	cmd.AddCommand(newKubectlListCommand(opts, runner))
	cmd.AddCommand(newKubectlUseCommand(opts, runner))
	cmd.AddCommand(newKubectlNamespaceCommand(opts, runner))

	return cmd
}

func newKubectlStatusCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show kubectl status and current context",
		RunE: func(cmd *cobra.Command, _ []string) error {
			runnerT := execx.TimeoutRunner{Runner: runner, Timeout: opts.Timeout}
			def, ok := tools.FindByID("kubectl")
			if !ok {
				return fmt.Errorf("kubectl tool definition not found")
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

func newKubectlCurrentCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show current kubectl context and namespace",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			a := kubectl.New(execx.TimeoutRunner{Runner: runner, Timeout: opts.Timeout})
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
			if v := strings.TrimSpace(cur["namespace"]); v != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "namespace: %s\n", v)
			}
			return nil
		},
	}
}

func newKubectlListCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List kubeconfig contexts",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			a := kubectl.New(execx.TimeoutRunner{Runner: runner, Timeout: opts.Timeout})

			contexts, warnings, errs, err := a.ListContexts(ctx)
			if err != nil {
				errs = append(errs, err.Error())
			}
			cur, _, _, _ := a.Current(ctx)
			currentName := strings.TrimSpace(cur["context"])
			currentNamespace := strings.TrimSpace(cur["namespace"])

			if opts.JSON {
				items := make([]map[string]any, 0, len(contexts))
				for _, name := range contexts {
					obj := map[string]any{
						"name":       name,
						"is_current": name == currentName,
					}
					if name == currentName && currentNamespace != "" {
						obj["namespace"] = currentNamespace
					}
					items = append(items, obj)
				}
				return output.PrintJSON(cmd.OutOrStdout(), map[string]any{
					"current":  cur,
					"contexts": items,
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

			for _, name := range contexts {
				mark := " "
				if name == currentName {
					mark = "*"
				}
				if name == currentName && currentNamespace != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "%s %s (namespace=%s)\n", mark, name, currentNamespace)
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", mark, name)
				}
			}
			return nil
		},
	}
}

func newKubectlUseCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	var namespace string

	cmd := &cobra.Command{
		Use:   "use <context>",
		Short: "Switch current kubectl context (and optionally set namespace for it)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			a := kubectl.New(execx.TimeoutRunner{Runner: runner, Timeout: opts.Timeout})

			contextName := args[0]
			if err := a.UseContext(ctx, contextName); err != nil {
				if opts.JSON {
					_ = output.PrintJSON(cmd.OutOrStdout(), model.UseResult{OK: false, ToolID: "kubectl", Error: err.Error()})
				}
				return err
			}
			if strings.TrimSpace(namespace) != "" {
				if err := a.SetNamespaceForContext(ctx, contextName, namespace); err != nil {
					if opts.JSON {
						_ = output.PrintJSON(cmd.OutOrStdout(), model.UseResult{OK: false, ToolID: "kubectl", Error: err.Error()})
					}
					return err
				}
			}

			cur, _, _, _ := a.Current(ctx)
			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), model.UseResult{OK: true, ToolID: "kubectl", Current: cur})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "switched kubectl context to %s\n", contextName)
			if strings.TrimSpace(namespace) != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "set namespace to %s\n", namespace)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&namespace, "namespace", "", "Namespace to set for the target context")
	return cmd
}

func newKubectlNamespaceCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "namespace <namespace>",
		Short: "Set default namespace for the current kubectl context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			a := kubectl.New(execx.TimeoutRunner{Runner: runner, Timeout: opts.Timeout})
			ns := args[0]
			if err := a.SetNamespaceForCurrentContext(ctx, ns); err != nil {
				if opts.JSON {
					_ = output.PrintJSON(cmd.OutOrStdout(), model.UseResult{OK: false, ToolID: "kubectl", Error: err.Error()})
				}
				return err
			}
			cur, _, _, _ := a.Current(ctx)
			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), model.UseResult{OK: true, ToolID: "kubectl", Current: cur})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "set current kubectl namespace to %s\n", ns)
			return nil
		},
	}
}
