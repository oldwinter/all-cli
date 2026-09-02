package cli

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
)

func TestWranglerStatusJSON(t *testing.T) {
	summary := model.ToolSummary{
		ID:              "wrangler",
		DisplayName:     "wrangler",
		Category:        "cloud",
		Installed:       true,
		InstallPath:     "/usr/bin/wrangler",
		ConfiguredState: model.ConfiguredYes,
		Configured:      true,
		Capabilities:    model.Capability{HasContexts: true},
		Current: map[string]string{
			"logged_in":      "yes",
			"accounts_count": "1",
		},
	}
	stubToolEvaluation(t, "wrangler", summary)

	opts := &rootOptions{JSON: true, Timeout: time.Second}
	stdout, stderr, err := executeTestCommand(t, newWranglerCommand(opts, cliFakeRunner{}), "status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var got model.ToolSummary
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if got.ID != "wrangler" || got.Current["logged_in"] != "yes" {
		t.Fatalf("unexpected summary: %#v", got)
	}
}

func TestWranglerStatusPlain(t *testing.T) {
	summary := model.ToolSummary{
		ID:              "wrangler",
		DisplayName:     "wrangler",
		Category:        "cloud",
		Installed:       true,
		ConfiguredState: model.ConfiguredUnknown,
		Capabilities:    model.Capability{HasContexts: true},
		Warnings:        []string{"multiple wrangler accounts detected; no single global default"},
	}
	stubToolEvaluation(t, "wrangler", summary)

	opts := &rootOptions{Timeout: time.Second}
	stdout, stderr, err := executeTestCommand(t, newWranglerCommand(opts, cliFakeRunner{}), "status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, needle := range []string{"TOOL", "wrangler", "Warnings:", "- wrangler: multiple wrangler accounts detected; no single global default"} {
		if !strings.Contains(stdout, needle) {
			t.Fatalf("expected %q in stdout, got:\n%s", needle, stdout)
		}
	}
}

func TestWranglerCurrentJSON(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"wrangler whoami --json": {
				Stdout: `{"loggedIn":true,"accounts":[{"id":"3ba1294bcdfb7a6f8c113ebc120411df"}]}`,
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newWranglerCommand(opts, runner), "current")
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
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if got.Current["logged_in"] != "yes" || got.Current["account_id"] != "3ba1294bcdfb7a6f8c113ebc120411df" {
		t.Fatalf("unexpected current payload: %#v", got.Current)
	}
}

func TestWranglerCurrentPlainPrintsDiagnostics(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"wrangler whoami --json": {
				Stdout: `{"loggedIn":true,"accounts":[{"id":"3ba1294bcdfb7a6f8c113ebc120411df"},{"id":"2371c3163e63aba96bd280648d9ffffc"}]}`,
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newWranglerCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, needle := range []string{"logged_in: yes", "accounts_count: 2"} {
		if !strings.Contains(stdout, needle) {
			t.Fatalf("expected %q in stdout, got:\n%s", needle, stdout)
		}
	}
	if !strings.Contains(stderr, "warning: multiple wrangler accounts detected; no single global default") {
		t.Fatalf("expected warning on stderr, got %q", stderr)
	}
}

func TestWranglerCurrentPlainReportsDeadlineErrors(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"wrangler whoami --json": {
				ExitCode: 1,
				Err:      context.DeadlineExceeded,
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newWranglerCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "error: context deadline exceeded") {
		t.Fatalf("expected deadline error in stderr, got %q", stderr)
	}
}

func TestWranglerCurrentJSONReportsDeadlineErrors(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"wrangler whoami --json": {
				ExitCode: 1,
				Err:      context.DeadlineExceeded,
			},
		},
	}

	stdout, _, err := executeTestCommand(t, newWranglerCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got struct {
		Errors []string `json:"errors"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if len(got.Errors) != 1 || got.Errors[0] != context.DeadlineExceeded.Error() {
		t.Fatalf("unexpected errors payload: %#v", got)
	}
}

func TestRootCommandIncludesWrangler(t *testing.T) {
	cmd := NewRootCommand()

	for _, child := range cmd.Commands() {
		if child.Name() == "wrangler" {
			return
		}
	}
	t.Fatal("expected root command to register wrangler")
}

func TestWranglerCurrentJSONMissingBinaryReportsError(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"wrangler whoami --json": {
				ExitCode: 1,
				Err:      exec.ErrNotFound,
			},
		},
	}

	stdout, _, err := executeTestCommand(t, newWranglerCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got struct {
		Current map[string]string `json:"current"`
		Errors  []string          `json:"errors"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if len(got.Errors) == 0 {
		t.Fatalf("expected missing-binary error, got %#v", got)
	}
	if got.Current["logged_in"] == "no" {
		t.Fatalf("missing binary must not report logged_in=no: %#v", got)
	}
}

func TestWranglerCurrentPlainMissingBinaryDoesNotPrintLoggedOut(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"wrangler whoami --json": {
				ExitCode: 1,
				Err:      exec.ErrNotFound,
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newWranglerCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(stdout, "logged_in: no") {
		t.Fatalf("missing binary must not print logged_in: no, got %q", stdout)
	}
	if !strings.Contains(stderr, "error:") {
		t.Fatalf("expected error on stderr, got %q", stderr)
	}
}
