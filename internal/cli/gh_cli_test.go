package cli

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
)

func TestGHStatusPlain(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"gh auth status --json hosts": {
				Stdout: `{"hosts":{"github.com":[{"login":"oldwinter","active":true,"state":"success","gitProtocol":"ssh","tokenSource":"oauth","scopes":"repo"}],"ghe.example.com":[{"login":"bot","active":true,"state":"success","gitProtocol":"https","tokenSource":"env"}]}}`,
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newGHCommand(opts, runner), "status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, needle := range []string{"github.com", "* oldwinter state=success git=ssh token=oauth scopes=repo", "ghe.example.com", "* bot state=success git=https token=env"} {
		if !strings.Contains(stdout, needle) {
			t.Fatalf("expected %q in stdout, got:\n%s", needle, stdout)
		}
	}
}

func TestGHStatusJSON(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"gh auth status --json hosts": {
				Stdout: `{"hosts":{"github.com":[{"login":"oldwinter","active":true,"state":"success","gitProtocol":"ssh","tokenSource":"oauth"}]}}`,
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newGHCommand(opts, runner), "status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	var got struct {
		Hosts []map[string]any `json:"hosts"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if len(got.Hosts) != 1 {
		t.Fatalf("unexpected payload: %#v", got)
	}
}

func TestGHStatusPlainError(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"gh auth status --json hosts": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "auth broken",
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newGHCommand(opts, runner), "status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	for _, needle := range []string{
		"error: auth broken",
		"error: gh auth status failed (exit=1)",
	} {
		if !strings.Contains(stderr, needle) {
			t.Fatalf("expected %q in stderr, got %q", needle, stderr)
		}
	}
}

func TestGHCurrentJSON(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"gh auth status --json hosts": {
				Stdout: `{"hosts":{"github.com":[{"login":"oldwinter","active":true,"state":"success","gitProtocol":"ssh","tokenSource":"oauth"}]}}`,
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newGHCommand(opts, runner), "current")
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
	if got.Current["hostname"] != "github.com" || got.Current["user"] != "oldwinter" {
		t.Fatalf("unexpected current: %#v", got.Current)
	}
}

func TestGHCurrentPlainWithWarning(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"gh auth status --json hosts": {
				Stdout: `{"hosts":{"github.com":[{"login":"oldwinter","active":true,"state":"success","gitProtocol":"ssh","tokenSource":"oauth"}],"ghe.example.com":[{"login":"bot","active":true,"state":"success","gitProtocol":"https","tokenSource":"env"}]}}`,
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newGHCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "github.com/oldwinter") {
		t.Fatalf("unexpected stdout:\n%s", stdout)
	}
	if !strings.Contains(stderr, "warning: multiple gh hosts configured; showing github.com") {
		t.Fatalf("expected warning in stderr, got %q", stderr)
	}
}

func TestGHCurrentPlainHostOnly(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"gh auth status --json hosts": {
				Stdout: `{"hosts":{"github.com":[{"login":"oldwinter","active":false,"state":"success","gitProtocol":"ssh","tokenSource":"oauth"}]}}`,
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newGHCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if strings.TrimSpace(stdout) != "github.com" {
		t.Fatalf("unexpected stdout: %q", stdout)
	}
}

func TestGHCurrentPlainError(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"gh auth status --json hosts": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "auth broken",
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newGHCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "error: auth broken") {
		t.Fatalf("expected stderr error, got %q", stderr)
	}
}

func TestGHUseJSONSuccess(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"gh auth switch --hostname github.com --user oldwinter": {},
			"gh auth status --json hosts": {
				Stdout: `{"hosts":{"github.com":[{"login":"oldwinter","active":true,"state":"success","gitProtocol":"ssh","tokenSource":"oauth"}]}}`,
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newGHCommand(opts, runner), "use", "--hostname", "github.com", "--user", "oldwinter")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var got model.UseResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if !got.OK || got.ToolID != "gh" || got.Current["user"] != "oldwinter" {
		t.Fatalf("unexpected use result: %#v", got)
	}
}

func TestGHUseJSONFailure(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"gh auth switch --hostname github.com --user oldwinter": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "permission denied",
			},
		},
	}

	stdout, _, err := executeTestCommand(t, newGHCommand(opts, runner), "use", "--hostname", "github.com", "--user", "oldwinter")
	if err == nil {
		t.Fatal("expected error")
	}
	var got model.UseResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if got.OK || !strings.Contains(got.Error, "permission denied") {
		t.Fatalf("unexpected use result: %#v", got)
	}
}

func TestGHUsePlainSuccess(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"gh auth switch --hostname github.com --user oldwinter": {},
			"gh auth status --json hosts": {
				Stdout: `{"hosts":{"github.com":[{"login":"oldwinter","active":true,"state":"success","gitProtocol":"ssh","tokenSource":"oauth"}]}}`,
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newGHCommand(opts, runner), "use", "--hostname", "github.com", "--user", "oldwinter")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "switched gh account to oldwinter@github.com") {
		t.Fatalf("unexpected stdout:\n%s", stdout)
	}
}
