package cli

import (
	"fmt"
	"strings"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/output"
	toolaliyun "github.com/oldwinter/all-cli/internal/tools/aliyun"
	"github.com/spf13/cobra"
)

func newAliyunCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "aliyun",
		Short: "Inspect Aliyun CLI profiles and current context",
	}

	cmd.AddCommand(newAliyunStatusCommand(opts, runner))
	cmd.AddCommand(newAliyunCurrentCommand(opts, runner))
	cmd.AddCommand(newAliyunListCommand(opts, runner))

	return cmd
}

func newAliyunStatusCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show Aliyun CLI status and current context",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runSingleToolStatusCommand(cmd, opts, runner, "aliyun")
		},
	}
}

func newAliyunCurrentCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show current Aliyun profile and fields",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			a := toolaliyun.New(execx.TimeoutRunner{Runner: runner, Timeout: opts.Timeout})
			cur, warnings, errs, err := a.Current(ctx)
			if err != nil {
				errs = append(errs, err.Error())
			}

			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), map[string]any{
					"current":  cur,
					"warnings": dedupeMessages(warnings),
					"errors":   dedupeMessages(errs),
				})
			}

			printDiagnostics(cmd.ErrOrStderr(), warnings, errs)
			for _, key := range []string{"profile", "region", "language", "valid"} {
				if value := strings.TrimSpace(cur[key]); value != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", key, value)
				}
			}
			return nil
		},
	}
}

func newAliyunListCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List Aliyun CLI profiles",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			a := toolaliyun.New(execx.TimeoutRunner{Runner: runner, Timeout: opts.Timeout})

			profiles, warnings, errs, err := a.ListProfiles(ctx)
			if err != nil {
				errs = append(errs, err.Error())
			}
			cur, moreWarnings, moreErrs, curErr := a.Current(ctx)
			warnings = append(warnings, moreWarnings...)
			errs = append(errs, moreErrs...)
			if curErr != nil {
				errs = append(errs, curErr.Error())
			}

			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), map[string]any{
					"current":  cur,
					"profiles": profiles,
					"warnings": dedupeMessages(warnings),
					"errors":   dedupeMessages(errs),
				})
			}

			printDiagnostics(cmd.ErrOrStderr(), warnings, errs)
			for _, profile := range profiles {
				mark := " "
				if profile.IsCurrent {
					mark = "*"
				}

				parts := []string{fmt.Sprintf("%s %s", mark, profile.Name)}
				if region := strings.TrimSpace(profile.Region); region != "" {
					parts = append(parts, "region="+region)
				}
				if language := strings.TrimSpace(profile.Language); language != "" {
					parts = append(parts, "language="+language)
				}
				if valid := strings.TrimSpace(profile.Valid); valid != "" {
					parts = append(parts, "valid="+valid)
				}
				fmt.Fprintln(cmd.OutOrStdout(), strings.Join(parts, " "))
			}
			return nil
		},
	}
}
