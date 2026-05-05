package cli

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
	"github.com/oldwinter/all-cli/internal/tools"
)

func TestDockerStatusJSON(t *testing.T) {
	summary := model.ToolSummary{
		ID:              "docker",
		DisplayName:     "docker",
		Category:        "containers",
		Installed:       true,
		InstallPath:     "/usr/bin/docker",
		ConfiguredState: model.ConfiguredYes,
		Configured:      true,
		Capabilities:    model.Capability{HasContexts: true, CanSwitch: true},
		Current:         map[string]string{"context": "prod"},
	}
	stubToolEvaluation(t, "docker", summary)

	opts := &rootOptions{JSON: true, Timeout: time.Second}
	stdout, stderr, err := executeTestCommand(t, newDockerCommand(opts, cliFakeRunner{}), "status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var got model.ToolSummary
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if got.ID != "docker" || got.Current["context"] != "prod" {
		t.Fatalf("unexpected summary: %#v", got)
	}
}

func TestDockerStatusPlain(t *testing.T) {
	summary := model.ToolSummary{
		ID:              "docker",
		DisplayName:     "docker",
		Category:        "containers",
		Installed:       true,
		ConfiguredState: model.ConfiguredUnknown,
		Warnings:        []string{"docker context \"staging\" error: unavailable"},
	}
	stubToolEvaluation(t, "docker", summary)

	opts := &rootOptions{Timeout: time.Second}
	stdout, stderr, err := executeTestCommand(t, newDockerCommand(opts, cliFakeRunner{}), "status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, needle := range []string{"TOOL", "docker", "Warnings:", "- docker: docker context \"staging\" error: unavailable"} {
		if !strings.Contains(stdout, needle) {
			t.Fatalf("expected %q in stdout, got:\n%s", needle, stdout)
		}
	}
}

func TestDockerCurrentJSON(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"docker context show": {Stdout: "prod\n"},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newDockerCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	var got struct {
		Current map[string]string `json:"current"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if got.Current["context"] != "prod" {
		t.Fatalf("unexpected current: %#v", got.Current)
	}
}

func TestDockerCurrentPlain(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"docker context show": {Stdout: "prod\n"},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newDockerCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if strings.TrimSpace(stdout) != "prod" {
		t.Fatalf("unexpected stdout: %q", stdout)
	}
}

func TestDockerCurrentPlainError(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"docker context show": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "docker unavailable",
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newDockerCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "error: docker unavailable") {
		t.Fatalf("expected stderr error, got %q", stderr)
	}
}

func TestDockerCurrentJSONError(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"docker context show": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "docker unavailable",
			},
		},
	}

	stdout, _, err := executeTestCommand(t, newDockerCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got struct {
		Errors []string `json:"errors"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if len(got.Errors) != 2 || got.Errors[0] != "docker unavailable" || !strings.Contains(got.Errors[1], "docker context show failed") {
		t.Fatalf("unexpected payload: %#v", got)
	}
}

func TestDockerListPlain(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"docker context ls --format {{json .}}": {
				Stdout: "{\"Current\":true,\"Name\":\"prod\",\"Description\":\"Prod cluster\"}\n{\"Current\":false,\"Name\":\"staging\",\"Description\":\"Staging cluster\"}\n",
			},
			"docker context show": {Stdout: "prod\n"},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newDockerCommand(opts, runner), "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, needle := range []string{"* prod\tProd cluster", "  staging\tStaging cluster"} {
		if !strings.Contains(stdout, needle) {
			t.Fatalf("expected %q in stdout, got:\n%s", needle, stdout)
		}
	}
}

func TestDockerListJSON(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"docker context ls --format {{json .}}": {
				Stdout: "{\"Current\":true,\"Name\":\"prod\",\"Description\":\"Prod cluster\"}\n{\"Current\":false,\"Name\":\"staging\",\"Description\":\"Staging cluster\"}\n",
			},
			"docker context show": {Stdout: "prod\n"},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newDockerCommand(opts, runner), "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	var got struct {
		Current  map[string]string `json:"current"`
		Contexts []map[string]any  `json:"contexts"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if got.Current["context"] != "prod" || len(got.Contexts) != 2 {
		t.Fatalf("unexpected payload: %#v", got)
	}
}

func TestDockerListPlainWithWarnings(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"docker context ls --format {{json .}}": {
				Stdout: "{\"Current\":true,\"Name\":\"prod\",\"Description\":\"Prod cluster\"}\n{\"Current\":false,\"Name\":\"staging\",\"Description\":\"Staging cluster\",\"Error\":\"unavailable\"}\n",
			},
			"docker context show": {Stdout: "prod\n"},
		},
	}

	_, stderr, err := executeTestCommand(t, newDockerCommand(opts, runner), "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stderr, `warning: docker context "staging" error: unavailable`) {
		t.Fatalf("expected warning in stderr, got %q", stderr)
	}
}

func TestDockerListPlainWithoutDescription(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"docker context ls --format {{json .}}": {
				Stdout: "{\"Current\":false,\"Name\":\"staging\",\"Description\":\"\"}\n",
			},
			"docker context show": {Stdout: "prod\n"},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newDockerCommand(opts, runner), "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "  staging\n") {
		t.Fatalf("unexpected stdout:\n%s", stdout)
	}
}

func TestDockerListJSONWithWarnings(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"docker context ls --format {{json .}}": {
				Stdout: "{\"Current\":false,\"Name\":\"staging\",\"Description\":\"\",\"Error\":\"unavailable\"}\n",
			},
			"docker context show": {Stdout: "prod\n"},
		},
	}

	stdout, _, err := executeTestCommand(t, newDockerCommand(opts, runner), "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got struct {
		Warnings []string `json:"warnings"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if len(got.Warnings) != 1 {
		t.Fatalf("unexpected payload: %#v", got)
	}
}

func TestDockerListPlainError(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"docker context ls --format {{json .}}": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "list failed",
			},
			"docker context show": {Stdout: "prod\n"},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newDockerCommand(opts, runner), "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	for _, needle := range []string{
		"error: list failed",
		"error: docker context ls failed (exit=1)",
	} {
		if !strings.Contains(stderr, needle) {
			t.Fatalf("expected %q in stderr, got %q", needle, stderr)
		}
	}
}

func TestDockerListJSONError(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"docker context ls --format {{json .}}": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "list failed",
			},
			"docker context show": {Stdout: "prod\n"},
		},
	}

	stdout, _, err := executeTestCommand(t, newDockerCommand(opts, runner), "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got struct {
		Errors []string `json:"errors"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if len(got.Errors) != 2 || got.Errors[0] != "list failed" {
		t.Fatalf("unexpected payload: %#v", got)
	}
}

func TestDockerUsePlainSuccess(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"docker context use prod": {},
			"docker context show":     {Stdout: "prod\n"},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newDockerCommand(opts, runner), "use", "prod")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "switched docker context to prod") {
		t.Fatalf("unexpected stdout:\n%s", stdout)
	}
}

func TestDockerUseJSONSuccess(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"docker context use prod": {},
			"docker context show":     {Stdout: "prod\n"},
		},
	}

	stdout, _, err := executeTestCommand(t, newDockerCommand(opts, runner), "use", "prod")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got model.UseResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if !got.OK || got.ToolID != "docker" || got.Current["context"] != "prod" {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestDockerUseJSONFailure(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"docker context use prod": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "context not found",
			},
		},
	}

	stdout, _, err := executeTestCommand(t, newDockerCommand(opts, runner), "use", "prod")
	if err == nil {
		t.Fatal("expected error")
	}
	var got model.UseResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if got.OK || !strings.Contains(got.Error, "context not found") {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestDockerFixRequiresDryRun(t *testing.T) {
	stubDockerStatusEvaluation(t, model.ToolSummary{ID: "docker", DisplayName: "docker", Category: "containers"})

	opts := &rootOptions{Timeout: time.Second}
	_, _, err := executeTestCommand(t, newDockerCommand(opts, cliFakeRunner{}), "fix")
	if err == nil || !strings.Contains(err.Error(), "--dry-run") {
		t.Fatalf("expected --dry-run error, got %v", err)
	}
}

func TestDockerFixDryRunJSON(t *testing.T) {
	stubDockerStatusEvaluation(t, model.ToolSummary{
		ID:              "docker",
		DisplayName:     "docker",
		Category:        "containers",
		Installed:       false,
		ConfiguredState: model.ConfiguredUnknown,
		Capabilities:    model.Capability{HasContexts: true, CanSwitch: true},
	})

	opts := &rootOptions{JSON: true, Timeout: time.Second}
	stdout, _, err := executeTestCommand(t, newDockerCommand(opts, cliFakeRunner{}), "fix", "--dry-run")
	if err != nil {
		t.Fatalf("docker fix --dry-run: %v", err)
	}
	var got model.FixPlan
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode fix plan: %v", err)
	}
	if !got.DryRun || got.Summary.Total != 1 || got.Items[0].RelatedTool != "docker" {
		t.Fatalf("unexpected plan: %#v", got)
	}
}

func TestDockerFixDryRunPlain(t *testing.T) {
	stubDockerStatusEvaluation(t, model.ToolSummary{
		ID:              "docker",
		DisplayName:     "docker",
		Category:        "containers",
		Installed:       true,
		ConfiguredState: model.ConfiguredNo,
	})

	opts := &rootOptions{Timeout: time.Second}
	stdout, _, err := executeTestCommand(t, newDockerCommand(opts, cliFakeRunner{}), "fix", "--dry-run")
	if err != nil {
		t.Fatalf("docker fix --dry-run: %v", err)
	}
	if !strings.Contains(stdout, "Fix plan: dry_run=true") || !strings.Contains(stdout, "docker") {
		t.Fatalf("unexpected stdout:\n%s", stdout)
	}
}

func TestDockerUpdateDryRunPlain(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"docker ps --format {{json .}}": {
				Stdout: "{\"ID\":\"1\",\"Image\":\"nginx:latest\",\"Names\":\"web\"}\n{\"ID\":\"2\",\"Image\":\"redis:7\",\"Names\":\"cache\"}\n",
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newDockerCommand(opts, runner), "update", "--dry-run")
	if err != nil {
		t.Fatalf("docker update --dry-run: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, needle := range []string{"Docker update plan (dry-run):", "docker pull nginx:latest", "containers: web", "docker pull redis:7"} {
		if !strings.Contains(stdout, needle) {
			t.Fatalf("expected %q in stdout:\n%s", needle, stdout)
		}
	}
}

func TestDockerUpdateDryRunJSON(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"docker ps --format {{json .}}": {
				Stdout: "{\"ID\":\"1\",\"Image\":\"nginx:latest\",\"Names\":\"web\"}\n",
			},
		},
	}

	stdout, _, err := executeTestCommand(t, newDockerCommand(opts, runner), "update", "--dry-run")
	if err != nil {
		t.Fatalf("docker update --dry-run: %v", err)
	}
	var got dockerUpdateResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode update result: %v", err)
	}
	if !got.DryRun || len(got.Updates) != 1 || got.Updates[0].Image != "nginx:latest" {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestDockerUpdateAllUsesAllContainers(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"docker ps -a --format {{json .}}": {
				Stdout: "{\"ID\":\"1\",\"Image\":\"nginx:latest\",\"Names\":\"web\"}\n",
			},
		},
	}

	stdout, _, err := executeTestCommand(t, newDockerCommand(opts, runner), "update", "--dry-run", "--all")
	if err != nil {
		t.Fatalf("docker update --dry-run --all: %v", err)
	}
	if !strings.Contains(stdout, "docker pull nginx:latest") {
		t.Fatalf("unexpected stdout:\n%s", stdout)
	}
}

func TestDockerUpdateExplicitImagesPullsWhenNotDryRun(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"docker pull nginx:latest": {},
			"docker pull redis:7":      {},
		},
	}

	stdout, _, err := executeTestCommand(t, newDockerCommand(opts, runner), "update", "--image", "redis:7", "--image", "nginx:latest")
	if err != nil {
		t.Fatalf("docker update --image: %v", err)
	}
	var got dockerUpdateResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode update result: %v", err)
	}
	if got.DryRun || len(got.Updates) != 2 || !got.Updates[0].Applied || !got.Updates[1].Applied {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestDockerUpdateSkipsUnsafeImageRefs(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	stdout, _, err := executeTestCommand(t, newDockerCommand(opts, cliFakeRunner{}), "update", "--dry-run", "--image", "sha256:abcdef", "--image", "nginx:latest")
	if err != nil {
		t.Fatalf("docker update --dry-run --image: %v", err)
	}
	var got dockerUpdateResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode update result: %v", err)
	}
	if len(got.Updates) != 1 || got.Updates[0].Image != "nginx:latest" || len(got.Warnings) == 0 {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestDockerUpdateReportsDockerErrors(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"docker ps --format {{json .}}": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "daemon unavailable",
			},
		},
	}

	stdout, _, err := executeTestCommand(t, newDockerCommand(opts, runner), "update", "--dry-run")
	if err != nil {
		t.Fatalf("json mode should report errors in payload, got %v", err)
	}
	var got dockerUpdateResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode update result: %v", err)
	}
	if len(got.Errors) != 2 || got.Errors[0] != "daemon unavailable" {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func stubDockerStatusEvaluation(t *testing.T, summary model.ToolSummary) {
	t.Helper()
	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "docker", DisplayName: "docker", Category: "containers", Binary: "docker"},
	})
	oldEvaluate := evaluateToolSummary
	evaluateToolSummary = func(_ context.Context, def tools.ToolDefinition, _ execx.Runner) model.ToolSummary {
		if def.ID != "docker" {
			t.Fatalf("unexpected tool %s", def.ID)
		}
		return summary
	}
	t.Cleanup(func() {
		evaluateToolSummary = oldEvaluate
	})
	stubShowStatusSpinner(t, false)
}
