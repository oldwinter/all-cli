package cli

import (
	"encoding/json"
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
			printDoctorReport(cmd.OutOrStdout(), report)
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
	return &cobra.Command{
		Use:   "diff <snapshot-a> <snapshot-b>",
		Short: "Diff two status snapshots",
		Long: `Diffs two status snapshots. Use - for either snapshot to read it from
standard input, which makes it possible to compare a saved snapshot with a live pipeline.
Standard input snapshots are limited to 1 MiB.`,
		Example: `  all-cli diff before.json after.json
  all-cli snapshot --json | all-cli diff before.json - --json`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
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
			report := diag.DiffSnapshots(before, after)
			if opts.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), report)
			}
			printSnapshotDiff(cmd.OutOrStdout(), report)
			return nil
		},
	}
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
	return report, nil
}

func printSnapshotDiff(w interface {
	Write([]byte) (int, error)
}, report model.SnapshotDiffReport) {
	fmt.Fprintf(w, "Snapshot diff: added=%d removed=%d changed=%d\n",
		report.Summary.Added,
		report.Summary.Removed,
		report.Summary.Changed,
	)
	if len(report.Changes) == 0 {
		fmt.Fprintln(w, "No changes.")
		return
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
		fmt.Fprintf(w, "- %s %s%s\n", change.ToolID, change.ChangeType, fields)
	}
}
