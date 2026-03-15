package cli

import (
	"fmt"
	"strings"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/output"
	toolaws "github.com/oldwinter/all-cli/internal/tools/aws"
	"github.com/spf13/cobra"
)

func newAWSCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "aws",
		Short: "Inspect AWS CLI profiles and current context",
	}

	cmd.AddCommand(newAWSStatusCommand(opts, runner))
	cmd.AddCommand(newAWSCurrentCommand(opts, runner))
	cmd.AddCommand(newAWSListCommand(opts, runner))

	return cmd
}

func newAWSStatusCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show AWS CLI status and current context",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runSingleToolStatusCommand(cmd, opts, runner, "aws")
		},
	}
}

func newAWSCurrentCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show current AWS profile, region, and output format",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			a := toolaws.New(execx.TimeoutRunner{Runner: runner, Timeout: opts.Timeout})
			cur, warnings, errs, _ := a.Current(ctx)

			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), map[string]any{
					"current":  cur,
					"warnings": dedupeMessages(warnings),
					"errors":   dedupeMessages(errs),
				})
			}

			printDiagnostics(cmd.ErrOrStderr(), warnings, errs)
			for _, key := range []string{"profile", "region", "output"} {
				if value := strings.TrimSpace(cur[key]); value != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", key, value)
				}
			}
			return nil
		},
	}
}

func newAWSListCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	type profileEntry struct {
		Name      string `json:"name"`
		IsCurrent bool   `json:"is_current"`
	}

	return &cobra.Command{
		Use:   "list",
		Short: "List AWS CLI profiles",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			a := toolaws.New(execx.TimeoutRunner{Runner: runner, Timeout: opts.Timeout})

			profiles, warnings, errs, err := a.ListProfiles(ctx)
			if err != nil {
				errs = append(errs, err.Error())
			}
			cur, moreWarnings, moreErrs, _ := a.Current(ctx)
			warnings = append(warnings, moreWarnings...)
			errs = append(errs, moreErrs...)

			currentProfile := strings.TrimSpace(cur["profile"])
			items := make([]profileEntry, 0, len(profiles))
			for _, profile := range profiles {
				items = append(items, profileEntry{
					Name:      profile,
					IsCurrent: profile == currentProfile,
				})
			}

			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), map[string]any{
					"current":  cur,
					"profiles": items,
					"warnings": dedupeMessages(warnings),
					"errors":   dedupeMessages(errs),
				})
			}

			printDiagnostics(cmd.ErrOrStderr(), warnings, errs)
			for _, item := range items {
				mark := " "
				if item.IsCurrent {
					mark = "*"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", mark, item.Name)
			}
			return nil
		},
	}
}
