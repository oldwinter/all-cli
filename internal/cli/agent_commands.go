package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	diag "github.com/oldwinter/all-cli/internal/diagnose"
	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
	"github.com/oldwinter/all-cli/internal/output"
	"github.com/spf13/cobra"
)

const maxStdinSnapshotBytes int64 = 1 << 20

var errSnapshotDifferences = errors.New("snapshot differences found")

func newDiagnoseCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	var toolsFilter string
	var profile string

	cmd := &cobra.Command{
		Use:   "diagnose",
		Short: "Generate agent-readable diagnostics from CLI status",
		Long: `Generates structured diagnostics from the same tool evaluation used by status.
Diagnostics include severity, evidence, suggested actions, autofix safety, and related tool IDs.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			report, err := buildDiagnosticReport(cmd, opts, runner, toolsFilter, profile)
			if err != nil {
				return err
			}
			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), report)
			}
			output.PrintDiagnosticReport(cmd.OutOrStdout(), report)
			return nil
		},
	}

	cmd.Flags().StringVar(&toolsFilter, "tools", "", "Comma-separated tool IDs to diagnose (e.g. kubectl,docker)")
	cmd.Flags().StringVar(&profile, "profile", diag.ProfileAgent, "Output profile: agent|human|ci")
	return cmd
}

func newDoctorCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	var toolsFilter string
	var profile string

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run read-only health checks for local CLI tools",
		RunE: func(cmd *cobra.Command, _ []string) error {
			report, err := buildDiagnosticReport(cmd, opts, runner, toolsFilter, profile)
			if err != nil {
				return err
			}
			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), report)
			}
			output.PrintDoctorReport(cmd.OutOrStdout(), report)
			return nil
		},
	}

	cmd.Flags().StringVar(&toolsFilter, "tools", "", "Comma-separated tool IDs to check (e.g. kubectl,docker)")
	cmd.Flags().StringVar(&profile, "profile", diag.ProfileHuman, "Output profile: agent|human|ci")
	return cmd
}

func newFixCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	var dryRun bool
	var toolsFilter string
	var profile string

	cmd := &cobra.Command{
		Use:   "fix",
		Short: "Preview safe diagnostic fixes",
		Long: `Builds a fix plan from diagnostics. The first implementation is dry-run only:
it does not run commands or mutate global CLI configuration.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !dryRun {
				return fmt.Errorf("fix currently requires --dry-run")
			}
			report, err := buildDiagnosticReport(cmd, opts, runner, toolsFilter, profile)
			if err != nil {
				return err
			}
			plan := diag.BuildFixPlan(report, diag.FixOptions{DryRun: true})
			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), plan)
			}
			output.PrintFixPlan(cmd.OutOrStdout(), plan)
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview fixes without running commands or changing configuration")
	cmd.Flags().StringVar(&toolsFilter, "tools", "", "Comma-separated tool IDs to plan fixes for")
	cmd.Flags().StringVar(&profile, "profile", diag.ProfileAgent, "Output profile: agent|human|ci")
	return cmd
}

func newSnapshotCommand(opts *rootOptions, runner execx.Runner) *cobra.Command {
	var toolsFilter string

	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "Capture a status snapshot for later diffing",
		RunE: func(cmd *cobra.Command, _ []string) error {
			report, err := buildStatusReport(cmd.Context(), runner, opts.Timeout, toolsFilter)
			if err != nil {
				return err
			}
			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), report)
			}
			output.PrintStatusTable(cmd.OutOrStdout(), report)
			return nil
		},
	}

	cmd.Flags().StringVar(&toolsFilter, "tools", "", "Comma-separated tool IDs to snapshot")
	return cmd
}

func newDiffCommand(opts *rootOptions) *cobra.Command {
	var exitCode bool
	var toolsFilter string

	cmd := &cobra.Command{
		Use:   "diff <snapshot-a> <snapshot-b>",
		Short: "Diff two status snapshots",
		Long: `Diffs two status snapshots. Use - for either snapshot to read it from
standard input, which makes it possible to compare a saved snapshot with a live pipeline.
Standard input snapshots are limited to 1 MiB. Add --exit-code to return status 1 when
the snapshots differ while still printing the complete report. Use --tools to compare
only selected tracked tools; the summary and exit code then reflect only those tools.`,
		Example: `  all-cli diff before.json after.json
  all-cli diff before.json after.json --exit-code
  all-cli diff before.json after.json --tools kubectl,docker --exit-code
  all-cli snapshot --json | all-cli diff before.json - --json`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var selected map[string]bool
			if strings.TrimSpace(toolsFilter) != "" {
				var err error
				selected, err = parseToolsFilter(toolsFilter)
				if err != nil {
					return err
				}
			}
			if args[0] == "-" && args[1] == "-" {
				return fmt.Errorf(`diff accepts "-" for only one snapshot`)
			}
			before, err := readStatusSnapshot(args[0], cmd.InOrStdin())
			if err != nil {
				return err
			}
			after, err := readStatusSnapshot(args[1], cmd.InOrStdin())
			if err != nil {
				return err
			}
			before.Tools = filterSnapshotTools(before.Tools, selected)
			after.Tools = filterSnapshotTools(after.Tools, selected)
			report := diag.DiffSnapshots(before, after)
			if opts.JSON {
				if err := output.PrintJSON(cmd.OutOrStdout(), report); err != nil {
					return err
				}
			} else {
				output.PrintSnapshotDiff(cmd.OutOrStdout(), report)
			}
			if exitCode && len(report.Changes) > 0 {
				cmd.Root().SilenceErrors = true
				return errSnapshotDifferences
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&exitCode, "exit-code", false, "Return status 1 when snapshots differ")
	cmd.Flags().StringVar(&toolsFilter, "tools", "", "Comma-separated tracked tool IDs to compare (e.g. kubectl,docker)")
	return cmd
}

func filterSnapshotTools(summaries []model.ToolSummary, selected map[string]bool) []model.ToolSummary {
	if selected == nil {
		return summaries
	}
	filtered := make([]model.ToolSummary, 0, len(summaries))
	for _, tool := range summaries {
		if selected[tool.ID] {
			filtered = append(filtered, tool)
		}
	}
	return filtered
}

func buildDiagnosticReport(cmd *cobra.Command, opts *rootOptions, runner execx.Runner, toolsFilter, profile string) (model.DiagnosticReport, error) {
	if err := validateAgentProfile(profile); err != nil {
		return model.DiagnosticReport{}, err
	}
	statusReport, err := buildStatusReport(cmd.Context(), runner, opts.Timeout, toolsFilter)
	if err != nil {
		return model.DiagnosticReport{}, err
	}
	return diag.Generate(statusReport, diag.Options{Profile: profile}), nil
}

func validateAgentProfile(profile string) error {
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case "", diag.ProfileAgent, diag.ProfileHuman, diag.ProfileCI:
		return nil
	default:
		return fmt.Errorf("invalid --profile value %q (allowed: agent, human, ci)", profile)
	}
}

func readStatusSnapshot(path string, stdin io.Reader) (model.StatusReport, error) {
	source := path
	var data []byte
	var err error
	if path == "-" {
		source = "stdin"
		data, err = io.ReadAll(io.LimitReader(stdin, maxStdinSnapshotBytes+1))
		if int64(len(data)) > maxStdinSnapshotBytes {
			return model.StatusReport{}, fmt.Errorf("snapshot stdin exceeds 1 MiB limit")
		}
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return model.StatusReport{}, fmt.Errorf("read snapshot %s: %w", source, err)
	}
	var report model.StatusReport
	if err := json.Unmarshal(data, &report); err != nil {
		return model.StatusReport{}, fmt.Errorf("parse snapshot %s: %w", source, err)
	}
	if strings.TrimSpace(report.SchemaVersion) == "" {
		return model.StatusReport{}, fmt.Errorf("parse snapshot %s: missing schema_version", source)
	}
	return report, nil
}
