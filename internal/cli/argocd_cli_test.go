package cli

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
)

const argocdContextsOutput = `CURRENT  NAME          SERVER
*        prod-admin    https://argocd.example.com
         dev-admin     https://argocd-dev.example.com
`

func TestArgoCDListJSON(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"argocd context": {
				Stdout: argocdContextsOutput,
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newArgoCDCommand(opts, runner), "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	var got struct {
		Current  map[string]string `json:"current"`
		Contexts []struct {
			Name      string `json:"name"`
			Server    string `json:"server"`
			IsCurrent bool   `json:"is_current"`
		} `json:"contexts"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if got.Current["context"] != "prod-admin" || len(got.Contexts) != 2 || !got.Contexts[0].IsCurrent {
		t.Fatalf("unexpected payload: %#v", got)
	}
}

func TestArgoCDStatusPlain(t *testing.T) {
	summary := model.ToolSummary{
		ID:              "argocd",
		DisplayName:     "argocd",
		Category:        "cicd",
		Installed:       true,
		ConfiguredState: model.ConfiguredYes,
		Configured:      true,
		Warnings:        []string{"server warning"},
	}
	stubToolEvaluation(t, "argocd", summary)

	opts := &rootOptions{Timeout: time.Second}
	stdout, stderr, err := executeTestCommand(t, newArgoCDCommand(opts, cliFakeRunner{}), "status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "- argocd: server warning") {
		t.Fatalf("unexpected stdout:\n%s", stdout)
	}
}

func TestArgoCDStatusJSON(t *testing.T) {
	summary := model.ToolSummary{
		ID:              "argocd",
		DisplayName:     "argocd",
		Category:        "cicd",
		Installed:       true,
		ConfiguredState: model.ConfiguredYes,
		Configured:      true,
		Current:         map[string]string{"context": "prod-admin"},
	}
	stubToolEvaluation(t, "argocd", summary)

	opts := &rootOptions{JSON: true, Timeout: time.Second}
	stdout, stderr, err := executeTestCommand(t, newArgoCDCommand(opts, cliFakeRunner{}), "status")
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
	if got.ID != "argocd" || got.Current["context"] != "prod-admin" {
		t.Fatalf("unexpected summary: %#v", got)
	}
}

func TestArgoCDCurrentPlain(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"argocd context": {
				Stdout: argocdContextsOutput,
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newArgoCDCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, needle := range []string{"context: prod-admin", "server: https://argocd.example.com"} {
		if !strings.Contains(stdout, needle) {
			t.Fatalf("expected %q in stdout, got:\n%s", needle, stdout)
		}
	}
}

func TestArgoCDCurrentJSON(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"argocd context": {
				Stdout: argocdContextsOutput,
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newArgoCDCommand(opts, runner), "current")
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
	if got.Current["context"] != "prod-admin" || got.Current["server"] != "https://argocd.example.com" {
		t.Fatalf("unexpected payload: %#v", got.Current)
	}
}

func TestArgoCDCurrentPlainWithWarnings(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"argocd context": {
				Stdout: "CURRENT  NAME  SERVER\nbroken-row\n* prod-admin https://argocd.example.com\n",
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newArgoCDCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "context: prod-admin") {
		t.Fatalf("unexpected stdout:\n%s", stdout)
	}
	if !strings.Contains(stderr, "warning: unexpected argocd context row format on line 2") {
		t.Fatalf("expected warning in stderr, got %q", stderr)
	}
}

func TestArgoCDCurrentPlainWithoutServer(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"argocd context": {
				Stdout: "CURRENT  NAME  SERVER\n* prod-admin\n",
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newArgoCDCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr == "" || !strings.Contains(stderr, "warning: unexpected argocd context row format") {
		t.Fatalf("expected warning in stderr, got %q", stderr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout when no valid current context, got %q", stdout)
	}
}

func TestArgoCDCurrentJSONWithoutCurrentContext(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"argocd context": {
				Stdout: "CURRENT  NAME  SERVER\n  dev-admin https://argocd-dev.example.com\n",
			},
		},
	}

	stdout, _, err := executeTestCommand(t, newArgoCDCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got struct {
		Current map[string]string `json:"current"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if got.Current != nil {
		t.Fatalf("expected nil current payload, got %#v", got.Current)
	}
}

func TestArgoCDCurrentJSONError(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"argocd context": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "context unavailable",
			},
		},
	}

	stdout, _, err := executeTestCommand(t, newArgoCDCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got struct {
		Errors []string `json:"errors"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if len(got.Errors) != 2 || got.Errors[0] != "context unavailable" {
		t.Fatalf("unexpected payload: %#v", got)
	}
}

func TestArgoCDCurrentPlainError(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"argocd context": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "context unavailable",
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newArgoCDCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	for _, needle := range []string{
		"error: context unavailable",
		"error: argocd context failed (exit=1)",
	} {
		if !strings.Contains(stderr, needle) {
			t.Fatalf("expected %q in stderr, got %q", needle, stderr)
		}
	}
}

func TestArgoCDListJSONWithWarnings(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"argocd context": {
				Stdout: "CURRENT  NAME  SERVER\nbroken-row\n* prod-admin https://argocd.example.com\n",
			},
		},
	}

	stdout, _, err := executeTestCommand(t, newArgoCDCommand(opts, runner), "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got struct {
		Warnings []string `json:"warnings"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if len(got.Warnings) != 1 || !strings.Contains(got.Warnings[0], "unexpected argocd context row format") {
		t.Fatalf("unexpected payload: %#v", got)
	}
}

func TestArgoCDListPlain(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"argocd context": {
				Stdout: argocdContextsOutput,
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newArgoCDCommand(opts, runner), "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, needle := range []string{"* prod-admin (server=https://argocd.example.com)", "  dev-admin (server=https://argocd-dev.example.com)"} {
		if !strings.Contains(stdout, needle) {
			t.Fatalf("expected %q in stdout, got:\n%s", needle, stdout)
		}
	}
}

func TestArgoCDListPlainWithWarnings(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"argocd context": {
				Stdout: "CURRENT  NAME  SERVER\nbroken-row\n* prod-admin https://argocd.example.com\n",
			},
		},
	}

	_, stderr, err := executeTestCommand(t, newArgoCDCommand(opts, runner), "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stderr, "warning: unexpected argocd context row format on line 2") {
		t.Fatalf("expected warning in stderr, got %q", stderr)
	}
}

func TestArgoCDListPlainError(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"argocd context": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "context unavailable",
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newArgoCDCommand(opts, runner), "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	for _, needle := range []string{
		"error: context unavailable",
		"error: argocd context failed (exit=1)",
	} {
		if !strings.Contains(stderr, needle) {
			t.Fatalf("expected %q in stderr, got %q", needle, stderr)
		}
	}
}

func TestArgoCDListJSONError(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"argocd context": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "context unavailable",
			},
		},
	}

	stdout, _, err := executeTestCommand(t, newArgoCDCommand(opts, runner), "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got struct {
		Errors []string `json:"errors"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if len(got.Errors) != 2 || got.Errors[0] != "context unavailable" {
		t.Fatalf("unexpected payload: %#v", got)
	}
}

func TestArgoCDUseJSONSuccess(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"argocd context prod-admin": {},
			"argocd context": {
				Stdout: argocdContextsOutput,
			},
		},
	}

	stdout, _, err := executeTestCommand(t, newArgoCDCommand(opts, runner), "use", "prod-admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got model.UseResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if !got.OK || got.ToolID != "argocd" || got.Current["context"] != "prod-admin" {
		t.Fatalf("unexpected use result: %#v", got)
	}
}

func TestArgoCDUseJSONFailure(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"argocd context prod-admin": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "login required",
			},
		},
	}

	stdout, _, err := executeTestCommand(t, newArgoCDCommand(opts, runner), "use", "prod-admin")
	if err == nil {
		t.Fatal("expected error")
	}
	var got model.UseResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if got.OK || !strings.Contains(got.Error, "login required") {
		t.Fatalf("unexpected use result: %#v", got)
	}
}

func TestArgoCDUsePlainSuccess(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"argocd context prod-admin": {},
			"argocd context": {
				Stdout: argocdContextsOutput,
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newArgoCDCommand(opts, runner), "use", "prod-admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "switched argocd context to prod-admin") {
		t.Fatalf("unexpected stdout:\n%s", stdout)
	}
}
