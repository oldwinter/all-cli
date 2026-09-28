package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
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
Diagnostics include severity, evidence, suggested actions, autofix safety, and related tool IDs.
Defaults to the agent profile. For a human-readable health-check table, see 'all-cli doctor'.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			report, err := buildDiagnosticReport(cmd, opts, runner, toolsFilter, profile)
			if err != nil {
				return err
			}
			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), report)
			}
			printDiagnosticReport(cmd.OutOrStdout(), report)
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
	var fix bool
	var fixOpts diag.DoctorFixOptions

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run health checks for local CLI tools and optionally install missing ones",
		Long: `Prints a human-readable health-check table for local CLI tools (default
--profile human). For agent-readable diagnostics, see 'all-cli diagnose'.

With --fix, doctor also installs missing tools that have a supported installer
(brew, npm, pipx, or go). Use --fix --dry-run to preview the install commands
first, and --tools to limit which tools are fixed. Each install command runs with
a 10 minute timeout. Rerun doctor afterwards to confirm the new state.`,
		Example: `  all-cli doctor
  all-cli doctor --fix --dry-run
  all-cli doctor --fix --tools gh,kubectl --installer brew`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !fix && (fixOpts.DryRun || cmd.Flags().Changed("installer")) {
				return fmt.Errorf("--dry-run and --installer require --fix")
			}
			installer, err := diag.NormalizeDoctorInstaller(fixOpts.Installer)
			if err != nil {
				return err
			}
			fixOpts.Installer = installer
			report, err := buildDiagnosticReport(cmd, opts, runner, toolsFilter, profile)
			if err != nil {
				return err
			}
			if !fix {
				if opts.JSON {
					return output.PrintJSON(cmd.OutOrStdout(), report)
				}
				printDoctorReport(cmd.OutOrStdout(), report)
				return nil
			}
			installRunner := execx.TimeoutRunner{Runner: runner, Timeout: doctorInstallTimeout}
			fixes := diag.RunDoctorFixes(cmd.Context(), installRunner, report, fixOpts, doctorLookPath)
			if opts.JSON {
				fixReport := model.DoctorFixReport{SchemaVersion: model.DoctorFixSchemaVersionV01, Report: report, Fixes: fixes}
				if err := output.PrintJSON(cmd.OutOrStdout(), fixReport); err != nil {
					return err
				}
			} else {
				printDoctorReport(cmd.OutOrStdout(), report)
				printDoctorFixes(cmd.OutOrStdout(), fixes)
			}
			return doctorFixError(fixes)
		},
	}

	cmd.Flags().StringVar(&toolsFilter, "tools", "", "Comma-separated tool IDs to check and fix (e.g. kubectl,docker)")
	cmd.Flags().StringVar(&profile, "profile", diag.ProfileHuman, "Output profile: agent|human|ci")
	cmd.Flags().BoolVar(&fix, "fix", false, "Install missing tools that have a supported installer")
	cmd.Flags().BoolVar(&fixOpts.DryRun, "dry-run", false, "With --fix, preview install commands without running them")
	cmd.Flags().StringVar(&fixOpts.Installer, "installer", diag.DoctorInstallerAuto, "With --fix, installer to use: auto|brew|npm|pipx|go")
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
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !dryRun {
				return fmt.Errorf("fix currently requires --dry-run (example: all-cli fix --dry-run)")
			}
			report, err := buildDiagnosticReport(cmd, opts, runner, toolsFilter, profile)
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
		Long: `Captures the current status report for later diffing. Pass --json to emit the
machine-readable snapshot format that 'all-cli diff' consumes; the default output is a
human-readable table that diff cannot parse.`,
		Example: `  all-cli snapshot --json > before.json
  all-cli snapshot --json > after.json
  all-cli diff before.json after.json
  all-cli snapshot --json | all-cli diff before.json -`,
		Args: cobra.NoArgs,
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
	var details bool
	var exitCode bool
	var toolsFilter string

	cmd := &cobra.Command{
		Use:   "diff <snapshot-a> <snapshot-b>",
		Short: "Diff two status snapshots",
		Long: `Diffs two status snapshots. Use - for either snapshot to read it from
standard input, which makes it possible to compare a saved snapshot with a live pipeline.
Standard input snapshots are limited to 1 MiB. Add --exit-code to return status 1 when
the snapshots differ while still printing the complete report. Use --tools to compare
only selected tracked tools; the summary and exit code then reflect only those tools.
Add --details to show before/after values for changed fields in text output.
JSON always includes full before/after tool summaries and ignores --details.`,
		Example: `  all-cli diff before.json after.json
  all-cli diff before.json after.json --details
  all-cli diff before.json after.json --exit-code
  all-cli diff before.json after.json --tools kubectl,docker --exit-code
  all-cli snapshot --json | all-cli diff before.json - --json`,
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 2 {
				return fmt.Errorf("want diff <snapshot-a> <snapshot-b> (example: all-cli snapshot --json > before.json)")
			}
			return nil
		},
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
			} else if err := printSnapshotDiff(cmd.OutOrStdout(), report, details); err != nil {
				return err
			}
			if exitCode && len(report.Changes) > 0 {
				cmd.Root().SilenceErrors = true
				return errSnapshotDifferences
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&details, "details", false, "Show before/after values for changed fields (ignored with --json)")
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

func printDiagnosticReport(w interface {
	Write([]byte) (int, error)
}, report model.DiagnosticReport) {
	fmt.Fprintf(w, "Diagnostics: total=%d info=%d warning=%d error=%d profile=%s\n",
		report.Summary.Total,
		report.Summary.Info,
		report.Summary.Warning,
		report.Summary.Error,
		report.Profile,
	)
	if len(report.Diagnostics) == 0 {
		fmt.Fprintln(w, "No diagnostics found.")
		fmt.Fprintln(w, "Next steps:")
		fmt.Fprintln(w, "  - Browse tracked tools: all-cli catalog")
		fmt.Fprintln(w, "  - Check install status: all-cli status")
		fmt.Fprintln(w, "  - Preview safe fixes: all-cli fix --dry-run")
		return
	}
	for _, item := range report.Diagnostics {
		fmt.Fprintf(w, "\n[%s] %s: %s\n", item.Severity, item.RelatedTool, item.Problem)
		for _, evidence := range item.Evidence {
			fmt.Fprintf(w, "  evidence: %s\n", evidence)
		}
		for _, action := range item.SuggestedActions {
			fmt.Fprintf(w, "  action: %s", action.ID)
			if strings.TrimSpace(action.Title) != "" {
				fmt.Fprintf(w, " - %s", action.Title)
			}
			if len(action.Command) > 0 {
				fmt.Fprintf(w, " (%s)", strings.Join(action.Command, " "))
			}
			fmt.Fprintln(w)
		}
		fmt.Fprintf(w, "  safe_to_autofix: %t\n", item.SafeToAutofix)
	}
}

func printDoctorReport(w interface {
	Write([]byte) (int, error)
}, report model.DiagnosticReport) {
	fmt.Fprintln(w, "Doctor")
	printDiagnosticReport(w, report)
}

func printFixPlan(w interface {
	Write([]byte) (int, error)
}, plan model.FixPlan) {
	fmt.Fprintf(w, "Fix plan: dry_run=%t total=%d supported=%d blocked=%d\n",
		plan.DryRun,
		plan.Summary.Total,
		plan.Summary.Supported,
		plan.Summary.Blocked,
	)
	if len(plan.Items) == 0 {
		fmt.Fprintln(w, "No fixes planned.")
		return
	}
	for _, item := range plan.Items {
		fmt.Fprintf(w, "- %s %s supported=%t will_run=%t reason=%s\n",
			item.RelatedTool,
			item.Action.ID,
			item.Supported,
			item.WillRun,
			item.Reason,
		)
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
	if report.SchemaVersion != model.SchemaVersionV01 {
		return model.StatusReport{}, fmt.Errorf("parse snapshot %s: unsupported schema_version %q (expected %q)", source, report.SchemaVersion, model.SchemaVersionV01)
	}
	return report, nil
}

func printSnapshotDiff(w io.Writer, report model.SnapshotDiffReport, details bool) error {
	if _, err := fmt.Fprintf(w, "Snapshot diff: added=%d removed=%d changed=%d\n",
		report.Summary.Added,
		report.Summary.Removed,
		report.Summary.Changed,
	); err != nil {
		return err
	}
	if len(report.Changes) == 0 {
		_, err := fmt.Fprintln(w, "No changes.")
		return err
	}
	changes := append([]model.SnapshotToolChange(nil), report.Changes...)
	sort.SliceStable(changes, func(i, j int) bool {
		return changes[i].ToolID < changes[j].ToolID
	})
	for _, change := range changes {
		fields := ""
		if len(change.Fields) > 0 {
			fields = " fields=" + strings.Join(change.Fields, ",")
		}
		if _, err := fmt.Fprintf(w, "- %s %s%s\n", change.ToolID, change.ChangeType, fields); err != nil {
			return err
		}
		if details {
			if err := printSnapshotChangeDetails(w, change); err != nil {
				return err
			}
		}
	}
	return nil
}
