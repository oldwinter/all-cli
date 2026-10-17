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

// ghStatusOutput is the `gh status` / `gh list` JSON payload. Warnings and
// Errors are always emitted as arrays (never omitted, never null) so a consumer
// can range over them unconditionally, matching the shared currentEnvelope
// contract used by the other per-tool commands.
type ghStatusOutput struct {
	Hosts    []gh.Host `json:"hosts"`
	Warnings []string  `json:"warnings"`
	Errors   []string  `json:"errors"`
}

func newGHStatusCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show gh auth status (hosts and accounts)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runGHHostsReport(cmd, opts, runner)
		},
	}
}

func newGHListCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List gh hosts and accounts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runGHHostsReport(cmd, opts, runner)
		},
	}
}

// runGHHostsReport is the shared body of `gh status` and `gh list`. Both
// commands stay independent values so neither mutates the other's metadata.
func runGHHostsReport(cmd *cobra.Command, opts *rootOptions, runner execx.Runner) error {
	ctx := cmd.Context()
	a := gh.New(execx.TimeoutRunner{Runner: runner, Timeout: opts.Timeout})
	st, warnings, errs, err := a.Status(ctx)
	if err != nil {
		errs = append(errs, err.Error())
	}

	if opts.JSON {
		return output.PrintJSON(cmd.OutOrStdout(), ghStatusOutput{
			Hosts:    st.Hosts,
			Warnings: nonNil(dedupeMessages(warnings)),
			Errors:   nonNil(dedupeMessages(errs)),
		})
	}

	printDiagnostics(cmd.ErrOrStderr(), warnings, errs)

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
}

func newGHCurrentCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show the selected primary gh host/user",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			a := gh.New(execx.TimeoutRunner{Runner: runner, Timeout: opts.Timeout})
			cur, warnings, errs, err := a.Current(ctx)
			if err != nil {
				errs = append(errs, err.Error())
			}
			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), newCurrentEnvelope(cur, warnings, errs))
			}
			printDiagnostics(cmd.ErrOrStderr(), warnings, errs)
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
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			a := gh.New(execx.TimeoutRunner{Runner: runner, Timeout: opts.Timeout})
			if err := a.UseAccount(ctx, hostname, user); err != nil {
				return reportUseFailure(cmd, opts, "gh", err)
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
