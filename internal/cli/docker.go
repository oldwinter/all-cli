package cli

import (
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	diag "github.com/oldwinter/all-cli/internal/diagnose"
	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
	"github.com/oldwinter/all-cli/internal/output"
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
	cmd.AddCommand(newDockerFixCommand(opts, runner))
	cmd.AddCommand(newDockerUpdateCommand(opts, runner))

	return cmd
}

func newDockerStatusCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return newToolStatusCommand(opts, runner, "docker", "Show docker status and current context")
}

func newDockerCurrentCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show current docker context",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			a := docker.New(execx.TimeoutRunner{Runner: runner, Timeout: opts.Timeout})
			cur, warnings, errs, err := a.Current(ctx)
			if err != nil {
				errs = append(errs, err.Error())
			}
			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), newCurrentEnvelope(cur, warnings, errs))
			}
			printDiagnostics(cmd.ErrOrStderr(), warnings, errs)
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
		Args:  cobra.NoArgs,
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
					"warnings": nonNil(dedupeMessages(warnings)),
					"errors":   nonNil(dedupeMessages(errs)),
				})
			}

			printDiagnostics(cmd.ErrOrStderr(), warnings, errs)
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
			for _, c := range contexts {
				mark := " "
				if c.Name == currentName || c.IsCurrent {
					mark = "*"
				}
				desc := strings.TrimSpace(c.Description)
				if desc != "" {
					fmt.Fprintf(w, "%s %s\t%s\n", mark, c.Name, desc)
				} else {
					fmt.Fprintf(w, "%s %s\n", mark, c.Name)
				}
			}
			return w.Flush()
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
				return reportUseFailure(cmd, opts, "docker", err)
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

func newDockerFixCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "fix",
		Short: "Preview Docker diagnostic fixes",
		Args:  cobra.NoArgs,
		Long: `Builds a Docker-only fix plan from diagnostics. This command is dry-run only:
it does not run Docker commands or mutate Docker configuration.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !dryRun {
				return fmt.Errorf("docker fix currently requires --dry-run")
			}
			report, err := buildDockerDiagnosticReport(cmd, runner, opts.Timeout)
			if err != nil {
				return err
			}
			plan := diag.BuildFixPlan(report, diag.FixOptions{DryRun: true})
			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), plan)
			}
			printFixPlan(cmd.OutOrStdout(), plan)
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview fixes without running commands or changing Docker configuration")
	return cmd
}

func newDockerUpdateCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	var dryRun bool
	var all bool
	var imageRefs []string

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Plan or run Docker image refreshes",
		Args:  cobra.NoArgs,
		Long: `Plans Docker image refreshes from running containers by default.
Use --all to include stopped containers or --image to target explicit image refs.
Without --dry-run, this command runs docker pull for planned image refs only; it does not stop,
recreate, prune, remove containers, or change Docker contexts.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			a := docker.New(execx.TimeoutRunner{Runner: runner, Timeout: opts.Timeout})
			updates, warnings, errs, err := a.BuildUpdatePlan(ctx, docker.UpdatePlanOptions{
				All:    all,
				Images: imageRefs,
			})
			if err != nil {
				errs = append(errs, err.Error())
			}

			if err == nil && !dryRun {
				for i := range updates {
					pullErr := a.PullImage(ctx, updates[i].Image)
					if pullErr != nil {
						updates[i].Error = pullErr.Error()
						errs = append(errs, pullErr.Error())
						continue
					}
					updates[i].Applied = true
				}
			}

			result := dockerUpdateResult{
				DryRun:   dryRun,
				Updates:  nonNilUpdates(updates),
				Warnings: dedupeMessages(warnings),
				Errors:   dedupeMessages(errs),
			}
			if opts.JSON {
				if printErr := output.PrintJSON(cmd.OutOrStdout(), result); printErr != nil {
					return printErr
				}
			} else {
				printDiagnostics(cmd.ErrOrStderr(), result.Warnings, result.Errors)
				if dryRun {
					fmt.Fprintln(cmd.OutOrStdout(), "Docker update plan (dry-run):")
				} else {
					fmt.Fprintln(cmd.OutOrStdout(), "Docker update results:")
				}
				if len(updates) == 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "No Docker image update candidates found.")
				}
				for _, update := range updates {
					status := "planned"
					if update.Applied {
						status = "pulled"
					}
					if update.Error != "" {
						status = "error"
					}
					line := fmt.Sprintf("- %s: %s", status, strings.Join(update.Command, " "))
					if len(update.SourceContainers) > 0 {
						line += fmt.Sprintf(" (containers: %s)", strings.Join(update.SourceContainers, ","))
					}
					if update.Error != "" {
						line += fmt.Sprintf(" error=%s", update.Error)
					}
					fmt.Fprintln(cmd.OutOrStdout(), line)
				}
			}
			// This is the only mutating command, so a partial failure must exit
			// non-zero in every output format. Both paths have already emitted
			// their payload above, so the diagnostics are never lost.
			if len(result.Errors) > 0 {
				cmd.Root().SilenceErrors = true
				return fmt.Errorf("docker update completed with errors")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview docker pull commands without running them")
	cmd.Flags().BoolVar(&all, "all", false, "Include stopped containers when planning image refreshes")
	cmd.Flags().StringArrayVar(&imageRefs, "image", nil, "Image reference to refresh; repeat to specify multiple images")
	return cmd
}

type dockerUpdateResult struct {
	DryRun   bool                 `json:"dry_run"`
	Updates  []docker.ImageUpdate `json:"updates"`
	Warnings []string             `json:"warnings"`
	Errors   []string             `json:"errors"`
}

// nonNilUpdates normalizes a nil update slice so a failed plan marshals as []
// instead of null. Machine consumers of `docker update --json` must be able to
// range over updates unconditionally, exactly like warnings and errors.
func nonNilUpdates(updates []docker.ImageUpdate) []docker.ImageUpdate {
	if updates == nil {
		return []docker.ImageUpdate{}
	}
	return updates
}

func buildDockerDiagnosticReport(cmd *cobra.Command, runner execx.Runner, timeout time.Duration) (model.DiagnosticReport, error) {
	report, err := buildStatusReport(cmd.Context(), runner, timeout, "docker")
	if err != nil {
		return model.DiagnosticReport{}, err
	}
	return diag.Generate(report, diag.Options{Profile: diag.ProfileAgent}), nil
}
