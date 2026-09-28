package diagnose

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
)

type recordingRunner struct {
	calls   [][]string
	results map[string]execx.CmdResult
}

func (r *recordingRunner) Run(_ context.Context, name string, args ...string) execx.CmdResult {
	call := append([]string{name}, args...)
	r.calls = append(r.calls, call)
	if res, ok := r.results[strings.Join(call, " ")]; ok {
		return res
	}
	return execx.CmdResult{Stdout: "ok"}
}

func lookPathSet(installed ...string) func(string) (string, error) {
	set := map[string]bool{}
	for _, name := range installed {
		set[name] = true
	}
	return func(name string) (string, error) {
		if set[name] {
			return "/usr/local/bin/" + name, nil
		}
		return "", errors.New("not found")
	}
}

func installDiagnostic(toolID string) model.DiagnosticItem {
	return model.DiagnosticItem{
		ID:          toolID + ".missing",
		Severity:    model.DiagnosticInfo,
		Problem:     toolID + " is not installed.",
		RelatedTool: toolID,
		SuggestedActions: []model.SuggestedAction{{
			ID:      "install_tool",
			Title:   "Install " + toolID,
			Mutates: true,
		}},
	}
}

func TestNormalizeDoctorInstaller(t *testing.T) {
	t.Parallel()

	for _, in := range []string{"", "auto", "brew", "npm", "pipx", "go", " Brew ", "AUTO"} {
		got, err := NormalizeDoctorInstaller(in)
		if err != nil {
			t.Errorf("NormalizeDoctorInstaller(%q) = %v", in, err)
			continue
		}
		want := strings.ToLower(strings.TrimSpace(in))
		if want == "" {
			want = DoctorInstallerAuto
		}
		if got != want {
			t.Errorf("NormalizeDoctorInstaller(%q) = %q, want %q", in, got, want)
		}
	}
	for _, in := range []string{"apt", "brew2", " Brewer "} {
		if _, err := NormalizeDoctorInstaller(in); err == nil {
			t.Errorf("NormalizeDoctorInstaller(%q) should fail", in)
		}
	}
}

func TestSelectDoctorInstallCommand(t *testing.T) {
	t.Parallel()

	t.Run("tool without recipe", func(t *testing.T) {
		cmd, reason := selectDoctorInstallCommand("nonexistent-tool", DoctorInstallerAuto, lookPathSet("brew"))
		if cmd.Name != "" || reason == "" {
			t.Fatalf("got cmd=%+v reason=%q", cmd, reason)
		}
	})

	t.Run("auto picks first available installer", func(t *testing.T) {
		cmd, reason := selectDoctorInstallCommand("wrangler", DoctorInstallerAuto, lookPathSet("brew"))
		if cmd.Installer != doctorInstallerBrew || cmd.Name != "brew" || reason != "" {
			t.Fatalf("got cmd=%+v reason=%q", cmd, reason)
		}
	})

	t.Run("auto prefers recipe order over PATH order", func(t *testing.T) {
		cmd, reason := selectDoctorInstallCommand("wrangler", DoctorInstallerAuto, lookPathSet("brew", "npm"))
		if cmd.Installer != doctorInstallerNPM || reason != "" {
			t.Fatalf("got cmd=%+v reason=%q", cmd, reason)
		}
	})

	t.Run("auto reports tried installers when none found", func(t *testing.T) {
		cmd, reason := selectDoctorInstallCommand("wrangler", DoctorInstallerAuto, lookPathSet())
		if cmd.Name == "" || !strings.Contains(reason, "no supported installer found in PATH") {
			t.Fatalf("got cmd=%+v reason=%q", cmd, reason)
		}
		if !strings.Contains(reason, "npm") || !strings.Contains(reason, "brew") {
			t.Fatalf("reason missing tried installers: %q", reason)
		}
	})

	t.Run("specific installer with recipe and binary", func(t *testing.T) {
		cmd, reason := selectDoctorInstallCommand("wrangler", "brew", lookPathSet("brew"))
		if cmd.Installer != doctorInstallerBrew || reason != "" {
			t.Fatalf("got cmd=%+v reason=%q", cmd, reason)
		}
	})

	t.Run("specific installer recipe but binary missing", func(t *testing.T) {
		cmd, reason := selectDoctorInstallCommand("wrangler", "brew", lookPathSet())
		if cmd.Installer != doctorInstallerBrew || !strings.Contains(reason, "not found in PATH") {
			t.Fatalf("got cmd=%+v reason=%q", cmd, reason)
		}
	})

	t.Run("specific installer with no recipe", func(t *testing.T) {
		cmd, reason := selectDoctorInstallCommand("fd", "npm", lookPathSet("npm"))
		if cmd.Name != "" || !strings.Contains(reason, "npm installer is not supported") {
			t.Fatalf("got cmd=%+v reason=%q", cmd, reason)
		}
	})
}

func TestRunDoctorFixesDryRun(t *testing.T) {
	t.Parallel()

	runner := &recordingRunner{}
	report := model.DiagnosticReport{
		Diagnostics: []model.DiagnosticItem{
			installDiagnostic("fd"),
			{ID: "other", Severity: model.DiagnosticWarning, RelatedTool: "gh"},
			installDiagnostic("fd"), // duplicate tool is deduped
		},
	}
	out := RunDoctorFixes(context.Background(), runner, report,
		DoctorFixOptions{DryRun: true, Installer: DoctorInstallerAuto}, lookPathSet("brew"))

	if len(runner.calls) != 0 {
		t.Fatalf("dry-run ran commands: %v", runner.calls)
	}
	if len(out.Items) != 1 {
		t.Fatalf("items = %+v", out.Items)
	}
	item := out.Items[0]
	if item.ToolID != "fd" || item.Status != model.DoctorFixDryRun || !item.Supported {
		t.Fatalf("item = %+v", item)
	}
	if !reflect.DeepEqual(item.Command, []string{"brew", "install", "fd"}) {
		t.Fatalf("command = %v", item.Command)
	}
	want := model.DoctorFixSummary{Total: 1, Supported: 1, DryRun: 1}
	if out.Summary != want {
		t.Fatalf("summary = %+v, want %+v", out.Summary, want)
	}
	if !out.DryRun {
		t.Fatal("DryRun flag not propagated")
	}
}

func TestRunDoctorFixesInstallsAndReportsFailures(t *testing.T) {
	t.Parallel()

	runner := &recordingRunner{
		results: map[string]execx.CmdResult{
			"brew install fzf": {ExitCode: 1, Stderr: "kaboom"},
		},
	}
	report := model.DiagnosticReport{
		Diagnostics: []model.DiagnosticItem{
			installDiagnostic("fd"),
			installDiagnostic("fzf"),
		},
	}
	out := RunDoctorFixes(context.Background(), runner, report,
		DoctorFixOptions{DryRun: false, Installer: "brew"}, lookPathSet("brew"))

	if len(runner.calls) != 2 {
		t.Fatalf("runner calls = %v", runner.calls)
	}
	if !reflect.DeepEqual(runner.calls[0], []string{"brew", "install", "fd"}) {
		t.Fatalf("first call = %v", runner.calls[0])
	}
	installed, failed := out.Items[0], out.Items[1]
	if installed.Status != model.DoctorFixInstalled || installed.ExitCode != 0 {
		t.Fatalf("installed item = %+v", installed)
	}
	if failed.Status != model.DoctorFixFailed || failed.ExitCode != 1 || !strings.Contains(failed.Reason, "kaboom") {
		t.Fatalf("failed item = %+v", failed)
	}
	want := model.DoctorFixSummary{Total: 2, Supported: 2, Installed: 1, Failed: 1}
	if out.Summary != want {
		t.Fatalf("summary = %+v, want %+v", out.Summary, want)
	}
}

func TestRunDoctorFixesSkipsUnsupportedAndMissingInstallers(t *testing.T) {
	t.Parallel()

	runner := &recordingRunner{}
	report := model.DiagnosticReport{
		Diagnostics: []model.DiagnosticItem{
			installDiagnostic("unknown-tool"),
			installDiagnostic("fd"),
		},
	}
	// fd supports brew only; brew is absent from PATH -> skipped with reason.
	out := RunDoctorFixes(context.Background(), runner, report,
		DoctorFixOptions{DryRun: true, Installer: DoctorInstallerAuto}, lookPathSet())

	if len(runner.calls) != 0 {
		t.Fatalf("runner calls = %v", runner.calls)
	}
	if len(out.Items) != 2 {
		t.Fatalf("items = %+v", out.Items)
	}
	for i, item := range out.Items {
		if item.Status != model.DoctorFixSkipped || item.Reason == "" {
			t.Fatalf("item %d = %+v", i, item)
		}
	}
	// unknown-tool has no recipe at all; fd has a recipe but its installer is
	// absent, which still counts as supported with a skip reason.
	if out.Items[0].Supported || !strings.Contains(out.Items[0].Reason, "no supported automatic installer") {
		t.Fatalf("unknown-tool item = %+v", out.Items[0])
	}
	if !out.Items[1].Supported || !strings.Contains(out.Items[1].Reason, "no supported installer found in PATH") {
		t.Fatalf("fd item = %+v", out.Items[1])
	}
	want := model.DoctorFixSummary{Total: 2, Supported: 1, Skipped: 2}
	if out.Summary != want {
		t.Fatalf("summary = %+v, want %+v", out.Summary, want)
	}
}

func TestIsInstallDiagnostic(t *testing.T) {
	t.Parallel()

	if !isInstallDiagnostic(installDiagnostic("fd")) {
		t.Fatal("install diagnostic not detected")
	}
	for _, item := range []model.DiagnosticItem{
		{},
		{SuggestedActions: []model.SuggestedAction{}},
		{SuggestedActions: []model.SuggestedAction{{ID: "configure_tool"}}},
	} {
		if isInstallDiagnostic(item) {
			t.Fatalf("false positive for %+v", item)
		}
	}
}

func TestDoctorInstallCommandCommand(t *testing.T) {
	t.Parallel()

	cmd := doctorInstallCommand{Installer: doctorInstallerBrew, Name: "brew", Args: []string{"install", "fd"}}
	if got := cmd.Command(); !reflect.DeepEqual(got, []string{"brew", "install", "fd"}) {
		t.Fatalf("Command() = %v", got)
	}
}
