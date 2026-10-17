package cli

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
	"github.com/spf13/cobra"
)

// executeTestCommandWithCobraErrors runs cmd without silencing cobra's own
// error printing, so a test can assert whether the CLI double-reports a
// failure to stderr after already emitting a JSON payload on stdout.
func executeTestCommandWithCobraErrors(t *testing.T, cmd *cobra.Command, args ...string) (string, string, error) {
	t.Helper()

	var stdout, stderr strings.Builder
	cmd.SilenceUsage = true
	cmd.SilenceErrors = false
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)

	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

// decodeSingleToolStatus decodes a per-tool `status --json` payload. That
// payload is a full StatusReport containing exactly one tool, so machine
// consumers always get schema_version and can pipe it into `all-cli diff`.
func decodeSingleToolStatus(t *testing.T, payload string) model.ToolSummary {
	t.Helper()

	var report model.StatusReport
	if err := json.Unmarshal([]byte(payload), &report); err != nil {
		t.Fatalf("decode json: %v\npayload:\n%s", err, payload)
	}
	if report.SchemaVersion == "" {
		t.Fatalf("per-tool status JSON must carry schema_version, payload:\n%s", payload)
	}
	if len(report.Tools) != 1 {
		t.Fatalf("expected exactly one tool in report, got %d\npayload:\n%s", len(report.Tools), payload)
	}
	return report.Tools[0]
}

// assertJSONFieldIsEmptyArray asserts that a JSON object field marshals as an
// empty array rather than null.
func assertJSONFieldIsEmptyArray(t *testing.T, payload, field string) {
	t.Helper()

	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(payload), &raw); err != nil {
		t.Fatalf("decode json: %v\npayload:\n%s", err, payload)
	}
	got, ok := raw[field]
	if !ok {
		t.Fatalf("missing %q in payload:\n%s", field, payload)
	}
	if strings.TrimSpace(string(got)) != "[]" {
		t.Fatalf("%q = %s, want []", field, strings.TrimSpace(string(got)))
	}
}

func TestPerToolCommandsRejectUnexpectedArgs(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{}

	tests := []struct {
		name string
		cmd  *cobra.Command
		args []string
	}{
		{"docker status", newDockerCommand(opts, runner), []string{"status", "extra"}},
		{"docker current", newDockerCommand(opts, runner), []string{"current", "extra"}},
		{"docker list", newDockerCommand(opts, runner), []string{"list", "extra"}},
		{"docker update", newDockerCommand(opts, runner), []string{"update", "extra"}},
		{"docker fix", newDockerCommand(opts, runner), []string{"fix", "extra"}},
		{"docker use", newDockerCommand(opts, runner), []string{"use", "prod", "extra"}},
		{"kubectl status", newKubectlCommand(opts, runner), []string{"status", "extra"}},
		{"kubectl current", newKubectlCommand(opts, runner), []string{"current", "extra"}},
		{"kubectl list", newKubectlCommand(opts, runner), []string{"list", "extra"}},
		{"kubectl use", newKubectlCommand(opts, runner), []string{"use", "prod", "extra"}},
		{"kubectl namespace", newKubectlCommand(opts, runner), []string{"namespace", "ns", "extra"}},
		{"gh status", newGHCommand(opts, runner), []string{"status", "extra"}},
		{"gh current", newGHCommand(opts, runner), []string{"current", "extra"}},
		{"gh list", newGHCommand(opts, runner), []string{"list", "extra"}},
		{"gh use", newGHCommand(opts, runner), []string{"use", "extra"}},
		{"glab status", newGLabCommand(opts, runner), []string{"status", "extra"}},
		{"glab current", newGLabCommand(opts, runner), []string{"current", "extra"}},
		{"glab list", newGLabCommand(opts, runner), []string{"list", "extra"}},
		{"glab use", newGLabCommand(opts, runner), []string{"use", "host", "extra"}},
		{"argocd status", newArgoCDCommand(opts, runner), []string{"status", "extra"}},
		{"argocd current", newArgoCDCommand(opts, runner), []string{"current", "extra"}},
		{"argocd list", newArgoCDCommand(opts, runner), []string{"list", "extra"}},
		{"argocd use", newArgoCDCommand(opts, runner), []string{"use", "ctx", "extra"}},
		{"kargo status", newKargoCommand(opts, runner), []string{"status", "extra"}},
		{"kargo current", newKargoCommand(opts, runner), []string{"current", "extra"}},
		{"kargo use", newKargoCommand(opts, runner), []string{"use", "project", "extra"}},
		{"aws status", newAWSCommand(opts, runner), []string{"status", "extra"}},
		{"aws current", newAWSCommand(opts, runner), []string{"current", "extra"}},
		{"aws list", newAWSCommand(opts, runner), []string{"list", "extra"}},
		{"aliyun status", newAliyunCommand(opts, runner), []string{"status", "extra"}},
		{"aliyun current", newAliyunCommand(opts, runner), []string{"current", "extra"}},
		{"aliyun list", newAliyunCommand(opts, runner), []string{"list", "extra"}},
		{"mise status", newMiseCommand(opts, runner), []string{"status", "extra"}},
		{"mise current", newMiseCommand(opts, runner), []string{"current", "extra"}},
		{"k9s status", newK9sCommand(opts, runner), []string{"status", "extra"}},
		{"k9s current", newK9sCommand(opts, runner), []string{"current", "extra"}},
		{"wrangler status", newWranglerCommand(opts, runner), []string{"status", "extra"}},
		{"wrangler current", newWranglerCommand(opts, runner), []string{"current", "extra"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, _, err := executeTestCommand(t, tt.cmd, tt.args...)
			if err == nil {
				t.Fatalf("expected error for %v, got stdout:\n%s", tt.args, stdout)
			}
		})
	}
}

func TestPerToolCurrentJSONWarningsIsEmptyArray(t *testing.T) {
	tests := []struct {
		name string
		cmd  *cobra.Command
		args []string
	}{
		{"docker", newDockerCommand(&rootOptions{JSON: true, Timeout: time.Second}, cliFakeRunner{
			results: map[string]execx.CmdResult{"docker context show": {Stdout: "prod\n"}},
		}), []string{"current"}},
		{"kubectl", newKubectlCommand(&rootOptions{JSON: true, Timeout: time.Second}, cliFakeRunner{
			results: map[string]execx.CmdResult{
				"kubectl config current-context":                                        {Stdout: "prod\n"},
				"kubectl config view --minify --output jsonpath={..namespace}{\"\\n\"}": {Stdout: "payments\n"},
			},
		}), []string{"current"}},
		{"kargo", newKargoCommand(&rootOptions{JSON: true, Timeout: time.Second}, cliFakeRunner{
			results: map[string]execx.CmdResult{"kargo config view": {Stdout: "apiAddress: https://kargo.example.com\n"}},
		}), []string{"current"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := executeTestCommand(t, tt.cmd, tt.args...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if stderr != "" {
				t.Fatalf("expected empty stderr, got %q", stderr)
			}
			assertJSONFieldIsEmptyArray(t, stdout, "warnings")
			assertJSONFieldIsEmptyArray(t, stdout, "errors")
		})
	}
}

func TestUseJSONFailureDoesNotWriteStderr(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}

	tests := []struct {
		name string
		cmd  *cobra.Command
		args []string
	}{
		{
			name: "docker",
			cmd: newDockerCommand(opts, cliFakeRunner{results: map[string]execx.CmdResult{
				"docker context use prod": {ExitCode: 1, Err: assertError("exit status 1"), Stderr: "context not found"},
			}}),
			args: []string{"use", "prod"},
		},
		{
			name: "kubectl",
			cmd: newKubectlCommand(opts, cliFakeRunner{results: map[string]execx.CmdResult{
				"kubectl config use-context prod": {ExitCode: 1, Err: assertError("exit status 1"), Stderr: "context switch failed"},
			}}),
			args: []string{"use", "prod"},
		},
		{
			name: "argocd",
			cmd: newArgoCDCommand(opts, cliFakeRunner{results: map[string]execx.CmdResult{
				"argocd context prod-admin": {ExitCode: 1, Err: assertError("exit status 1"), Stderr: "login required"},
			}}),
			args: []string{"use", "prod-admin"},
		},
		{
			name: "gh",
			cmd: newGHCommand(opts, cliFakeRunner{results: map[string]execx.CmdResult{
				"gh auth switch --hostname github.com --user oldwinter": {ExitCode: 1, Err: assertError("exit status 1"), Stderr: "permission denied"},
			}}),
			args: []string{"use", "--hostname", "github.com", "--user", "oldwinter"},
		},
		{
			name: "glab",
			cmd: newGLabCommand(opts, cliFakeRunner{results: map[string]execx.CmdResult{
				"glab config set host gitlab.example.com": {ExitCode: 1, Err: assertError("exit status 1"), Stderr: "write failed"},
			}}),
			args: []string{"use", "gitlab.example.com"},
		},
		{
			name: "kargo",
			cmd: newKargoCommand(opts, cliFakeRunner{results: map[string]execx.CmdResult{
				"kargo config set-project payments": {ExitCode: 1, Err: assertError("exit status 1"), Stderr: "write failed"},
			}}),
			args: []string{"use", "payments"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := executeTestCommandWithCobraErrors(t, tt.cmd, tt.args...)
			if err == nil {
				t.Fatal("expected error")
			}
			// The JSON failure envelope must be the only report: no second
			// "Error: ..." line from cobra on stderr.
			if stderr != "" {
				t.Fatalf("expected empty stderr, got %q", stderr)
			}
			var got struct {
				OK    bool   `json:"ok"`
				Error string `json:"error"`
			}
			if err := json.Unmarshal([]byte(stdout), &got); err != nil {
				t.Fatalf("decode json: %v\npayload:\n%s", err, stdout)
			}
			if got.OK || got.Error == "" {
				t.Fatalf("unexpected failure envelope: %#v", got)
			}
		})
	}
}

func TestGHAndGLabAliasesAreIndependentCommands(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{}

	ghStatus := newGHStatusCommand(opts, runner)
	ghList := newGHListCommand(opts, runner)
	if ghStatus == ghList {
		t.Fatal("gh status and gh list must not share one *cobra.Command value")
	}
	if ghStatus.Use != "status" || ghList.Use != "list" {
		t.Fatalf("unexpected Use values: status=%q list=%q", ghStatus.Use, ghList.Use)
	}
	if ghStatus.Short == ghList.Short {
		t.Fatalf("expected distinct Short strings, both %q", ghStatus.Short)
	}

	glabStatus := newGLabStatusCommand(opts, runner)
	glabCurrent := newGLabCurrentCommand(opts, runner)
	if glabStatus == glabCurrent {
		t.Fatal("glab status and glab current must not share one *cobra.Command value")
	}
	if glabStatus.Use != "status" || glabCurrent.Use != "current" {
		t.Fatalf("unexpected Use values: status=%q current=%q", glabStatus.Use, glabCurrent.Use)
	}
	if glabStatus.Short == glabCurrent.Short {
		t.Fatalf("expected distinct Short strings, both %q", glabStatus.Short)
	}
}
