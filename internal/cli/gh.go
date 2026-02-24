package cli

import (
	"fmt"
	"strings"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
	"github.com/oldwinter/all-cli/internal/output"
	"github.com/oldwinter/all-cli/internal/tools/gh"
	"github.com/spf13/cobra"
)

func newGHCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gh",
		Short: "Inspect/switch gh auth context",
	}

	cmd.AddCommand(newGHStatusCommand(opts, runner))
	cmd.AddCommand(newGHCurrentCommand(opts, runner))
	cmd.AddCommand(newGHListCommand(opts, runner))
	cmd.AddCommand(newGHUseCommand(opts, runner))

	return cmd
}

type ghStatusOutput struct {
	Hosts    []gh.Host `json:"hosts"`
	Warnings []string  `json:"warnings,omitempty"`
	Errors   []string  `json:"errors,omitempty"`
}

func newGHStatusCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show gh auth status (hosts and accounts)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			a := gh.New(execx.TimeoutRunner{Runner: runner, Timeout: opts.Timeout})
			st, warnings, errs, err := a.Status(ctx)
			if err != nil {
				errs = append(errs, err.Error())
			}

			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), ghStatusOutput{Hosts: st.Hosts, Warnings: warnings, Errors: errs})
			}

			for _, w := range warnings {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", w)
			}
			for _, e := range errs {
				fmt.Fprintf(cmd.ErrOrStderr(), "error: %s\n", e)
			}

			for _, h := range st.Hosts {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\n", h.Hostname)
				for _, acc := range h.Accounts {
					mark := " "
					if acc.Active {
						mark = "*"
					}
					scopes := ""
					if strings.TrimSpace(acc.Scopes) != "" {
						scopes = fmt.Sprintf(" scopes=%s", acc.Scopes)
					}
					fmt.Fprintf(cmd.OutOrStdout(), "  %s %s state=%s git=%s token=%s%s\n",
						mark,
						acc.Login,
						acc.State,
						acc.GitProtocol,
						acc.TokenSource,
						scopes,
					)
				}
			}
			return nil
		},
	}
}

func newGHListCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	cmd := newGHStatusCommand(opts, runner)
	cmd.Use = "list"
	cmd.Short = "List gh hosts and accounts"
	return cmd
}

func newGHCurrentCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show the selected primary gh host/user",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			a := gh.New(execx.TimeoutRunner{Runner: runner, Timeout: opts.Timeout})
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
			h := strings.TrimSpace(cur["hostname"])
			u := strings.TrimSpace(cur["user"])
			if h != "" && u != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "%s/%s\n", h, u)
				return nil
			}
			if h != "" {
				fmt.Fprintln(cmd.OutOrStdout(), h)
			}
			return nil
		},
	}
}

func newGHUseCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	var hostname string
	var user string

	cmd := &cobra.Command{
		Use:   "use",
		Short: "Switch the active gh account for a host",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			a := gh.New(execx.TimeoutRunner{Runner: runner, Timeout: opts.Timeout})
			if err := a.UseAccount(ctx, hostname, user); err != nil {
				if opts.JSON {
					_ = output.PrintJSON(cmd.OutOrStdout(), model.UseResult{OK: false, ToolID: "gh", Error: err.Error()})
				}
				return err
			}
			cur, _, _, _ := a.Current(ctx)
			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), model.UseResult{OK: true, ToolID: "gh", Current: cur})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "switched gh account to %s@%s\n", user, hostname)
			return nil
		},
	}

	cmd.Flags().StringVar(&hostname, "hostname", "", "GitHub host (e.g. github.com)")
	cmd.Flags().StringVar(&user, "user", "", "User/login to switch to")
	_ = cmd.MarkFlagRequired("hostname")
	_ = cmd.MarkFlagRequired("user")

	return cmd
}
