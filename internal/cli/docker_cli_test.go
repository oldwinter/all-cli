package cli

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
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
