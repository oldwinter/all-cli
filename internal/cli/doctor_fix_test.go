package cli

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
	"github.com/oldwinter/all-cli/internal/tools"
)

type recordingInstallRunner struct {
	mu      sync.Mutex
	calls   []string
	results map[string]execx.CmdResult
}

func (r *recordingInstallRunner) Run(ctx context.Context, name string, args ...string) execx.CmdResult {
	key := strings.Join(append([]string{name}, args...), " ")
	r.mu.Lock()
	r.calls = append(r.calls, key)
	r.mu.Unlock()
	if _, ok := ctx.Deadline(); !ok {
		return execx.CmdResult{ExitCode: 1, Err: errors.New("install ran without a timeout")}
	}
	if res, ok := r.results[key]; ok {
		return res
	}
	return execx.CmdResult{}
}

// stubDoctorFixEnvironment reports gh, codex, and linear as missing and
// kubectl as installed; installers lists the installer binaries found in PATH.
func stubDoctorFixEnvironment(t *testing.T, installers ...string) {
	t.Helper()

	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "gh", DisplayName: "GitHub CLI", Category: "git", Binary: "gh"},
		{ID: "codex", DisplayName: "Codex CLI", Category: "ai", Binary: "codex"},
		{ID: "linear", DisplayName: "Linear CLI", Category: "work", Binary: "linear"},
		{ID: "kubectl", DisplayName: "kubectl", Category: "k8s", Binary: "kubectl"},
	})
	oldEvaluate := evaluateToolSummary
	evaluateToolSummary = func(_ context.Context, def tools.ToolDefinition, _ execx.Runner) model.ToolSummary {
		return model.ToolSummary{
			ID:              def.ID,
			DisplayName:     def.DisplayName,
			Category:        def.Category,
			Installed:       def.ID == "kubectl",
			ConfiguredState: model.ConfiguredNA,
		}
	}
	available := map[string]bool{}
	for _, name := range installers {
		available[name] = true
	}
	oldLookPath := doctorLookPath
	doctorLookPath = func(name string) (string, error) {
		if available[name] {
			return "/usr/local/bin/" + name, nil
		}
		return "", errors.New("not found")
	}
	t.Cleanup(func() {
		evaluateToolSummary = oldEvaluate
		doctorLookPath = oldLookPath
	})
	stubShowStatusSpinner(t, false)
}

func TestDoctorFixJSON(t *testing.T) {
	brewGH := []string{"brew", "install", "gh"}
	npmCodex := []string{"npm", "install", "-g", "@openai/codex"}
	linearSkipped := model.DoctorFixItem{ToolID: "linear", Status: model.DoctorFixSkipped, Reason: "no supported automatic installer for this tool"}

	tests := []struct {
		name       string
		args       []string
		installers []string
		results    map[string]execx.CmdResult
		wantCalls  []string
		wantItems  []model.DoctorFixItem
		wantErr    string
	}{
		{
			name:       "dry run previews without running",
			args:       []string{"--fix", "--dry-run"},
			installers: []string{"brew", "npm"},
			wantItems: []model.DoctorFixItem{
				{ToolID: "codex", Installer: "npm", Command: npmCodex, Supported: true, Status: model.DoctorFixDryRun},
				{ToolID: "gh", Installer: "brew", Command: brewGH, Supported: true, Status: model.DoctorFixDryRun},
				linearSkipped,
			},
		},
		{
			name:       "tools filter scopes fixes",
			args:       []string{"--fix", "--tools", "gh"},
			installers: []string{"brew", "npm"},
			wantCalls:  []string{"brew install gh"},
			wantItems: []model.DoctorFixItem{
				{ToolID: "gh", Installer: "brew", Command: brewGH, Supported: true, Status: model.DoctorFixInstalled},
			},
		},
		{
			name:       "failed install reports error",
			args:       []string{"--fix", "--tools", "gh,codex"},
			installers: []string{"brew", "npm"},
			results: map[string]execx.CmdResult{
				"npm install -g @openai/codex": {ExitCode: 243, Err: errors.New("exit status 243"), Stderr: "npm ERR! code EACCES\n"},
			},
			wantCalls: []string{"npm install -g @openai/codex", "brew install gh"},
			wantItems: []model.DoctorFixItem{
				{ToolID: "codex", Installer: "npm", Command: npmCodex, Supported: true, Status: model.DoctorFixFailed, Reason: "exit status 243: npm ERR! code EACCES", ExitCode: 243},
				{ToolID: "gh", Installer: "brew", Command: brewGH, Supported: true, Status: model.DoctorFixInstalled},
			},
			wantErr: "1 install command(s) failed",
		},
		{
			name:       "installer unsupported for tool",
			args:       []string{"--fix", "--installer", "pipx", "--tools", "gh"},
			installers: []string{"brew", "pipx"},
			wantItems: []model.DoctorFixItem{
				{ToolID: "gh", Status: model.DoctorFixSkipped, Reason: "pipx installer is not supported for this tool"},
			},
		},
		{
			name:       "explicit installer missing from PATH",
			args:       []string{"--fix", "--installer", "brew", "--tools", "gh"},
			installers: []string{"npm"},
			wantItems: []model.DoctorFixItem{
				{ToolID: "gh", Installer: "brew", Command: brewGH, Supported: true, Status: model.DoctorFixSkipped, Reason: "installer brew was not found in PATH"},
			},
		},
		{
			name:       "auto falls back to next installer in PATH",
			args:       []string{"--fix", "--dry-run", "--tools", "gh,codex"},
			installers: []string{"npm"},
			wantItems: []model.DoctorFixItem{
				{ToolID: "codex", Installer: "npm", Command: npmCodex, Supported: true, Status: model.DoctorFixDryRun},
				{ToolID: "gh", Installer: "brew", Command: brewGH, Supported: true, Status: model.DoctorFixSkipped, Reason: "no supported installer found in PATH (tried: brew)"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubDoctorFixEnvironment(t, tt.installers...)
			runner := &recordingInstallRunner{results: tt.results}

			opts := &rootOptions{JSON: true, Timeout: time.Second}
			stdout, _, err := executeTestCommand(t, newDoctorCommand(opts, runner), tt.args...)
			if tt.wantErr == "" && err != nil {
				t.Fatalf("doctor: %v", err)
			}
			if tt.wantErr != "" && (err == nil || err.Error() != tt.wantErr) {
				t.Fatalf("doctor error = %v, want %q", err, tt.wantErr)
			}

			var got model.DoctorFixReport
			if err := json.Unmarshal([]byte(stdout), &got); err != nil {
				t.Fatalf("decode doctor fix json: %v\n%s", err, stdout)
			}
			if got.SchemaVersion != model.DoctorFixSchemaVersionV01 {
				t.Fatalf("schema_version = %q", got.SchemaVersion)
			}
			if got.Report.SchemaVersion != model.DiagnosticSchemaVersionV01 {
				t.Fatalf("report schema_version = %q", got.Report.SchemaVersion)
			}
			if !reflect.DeepEqual(got.Fixes.Items, tt.wantItems) {
				t.Fatalf("items = %#v\nwant %#v", got.Fixes.Items, tt.wantItems)
			}
			if got.Fixes.Summary != summarizeDoctorFixes(tt.wantItems) {
				t.Fatalf("summary = %#v", got.Fixes.Summary)
			}
			if !reflect.DeepEqual(runner.calls, tt.wantCalls) {
				t.Fatalf("runner calls = %q, want %q", runner.calls, tt.wantCalls)
			}
		})
	}
}

func TestDoctorFixFlagValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "dry run requires fix", args: []string{"--dry-run"}, wantErr: "--dry-run and --installer require --fix"},
		{name: "installer requires fix", args: []string{"--installer", "brew"}, wantErr: "--dry-run and --installer require --fix"},
		{name: "unknown installer", args: []string{"--fix", "--installer", "apt"}, wantErr: `invalid --installer value "apt" (allowed: auto, brew, npm, pipx, go)`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubDoctorFixEnvironment(t, "brew", "npm")
			runner := &recordingInstallRunner{}

			stdout, _, err := executeTestCommand(t, newDoctorCommand(&rootOptions{Timeout: time.Second}, runner), tt.args...)
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("doctor error = %v, want %q", err, tt.wantErr)
			}
			if stdout != "" || len(runner.calls) != 0 {
				t.Fatalf("expected no output or installs, got stdout=%q calls=%q", stdout, runner.calls)
			}
		})
	}
}

func TestDoctorFixPlainOutputListsFixes(t *testing.T) {
	stubDoctorFixEnvironment(t, "brew", "npm")

	stdout, _, err := executeTestCommand(t, newDoctorCommand(&rootOptions{Timeout: time.Second}, &recordingInstallRunner{}), "--fix", "--dry-run")
	if err != nil {
		t.Fatalf("doctor: %v", err)
	}
	for _, needle := range []string{
		"Doctor",
		"Fixes",
		"installer=auto dry_run=true total=3 installed=0 preview=2 skipped=1 failed=0",
		"gh             preview   brew install gh",
		"linear         skipped   no supported automatic installer for this tool",
	} {
		if !strings.Contains(stdout, needle) {
			t.Fatalf("stdout missing %q:\n%s", needle, stdout)
		}
	}
}

func TestDoctorWithoutFixKeepsDiagnosticReport(t *testing.T) {
	stubDoctorFixEnvironment(t, "brew")
	runner := &recordingInstallRunner{}

	stdout, _, err := executeTestCommand(t, newDoctorCommand(&rootOptions{JSON: true, Timeout: time.Second}, runner))
	if err != nil {
		t.Fatalf("doctor: %v", err)
	}
	var got model.DiagnosticReport
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode doctor json: %v", err)
	}
	if got.SchemaVersion != model.DiagnosticSchemaVersionV01 || len(runner.calls) != 0 {
		t.Fatalf("expected read-only diagnostic report, got schema=%q calls=%q", got.SchemaVersion, runner.calls)
	}
}

func TestDoctorCommandFailureFallsBackToStdoutAndExitCode(t *testing.T) {
	tests := []struct {
		name string
		res  execx.CmdResult
		want string
	}{
		{name: "stdout", res: execx.CmdResult{ExitCode: 1, Stdout: "  Error:\n  formula not found  "}, want: "Error: formula not found"},
		{name: "exit code", res: execx.CmdResult{ExitCode: 7}, want: "command exited with code 7"},
		{name: "truncated", res: execx.CmdResult{ExitCode: 1, Stderr: strings.Repeat("x", 300)}, want: strings.Repeat("x", 237) + "..."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := doctorCommandFailure(tt.res); got != tt.want {
				t.Fatalf("doctorCommandFailure() = %q, want %q", got, tt.want)
			}
		})
	}
}
