package cli

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
)

func TestAWSStatusJSON(t *testing.T) {
	summary := model.ToolSummary{
		ID:              "aws",
		DisplayName:     "AWS CLI",
		Category:        "cloud",
		Installed:       true,
		InstallPath:     "/usr/bin/aws",
		ConfiguredState: model.ConfiguredYes,
		Configured:      true,
		Capabilities:    model.Capability{HasContexts: true},
		Current: map[string]string{
			"profile": "prod",
			"region":  "us-west-2",
		},
		Warnings: []string{"multiple profiles detected"},
		Errors:   []string{"cached error"},
	}
	stubToolEvaluation(t, "aws", summary)

	opts := &rootOptions{JSON: true, Timeout: time.Second}
	stdout, stderr, err := executeTestCommand(t, newAWSCommand(opts, cliFakeRunner{}), "status")
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
	if got.ID != "aws" || got.Current["profile"] != "prod" {
		t.Fatalf("unexpected summary: %#v", got)
	}
	if len(got.Warnings) != 1 || got.Warnings[0] != "multiple profiles detected" {
		t.Fatalf("unexpected warnings: %#v", got.Warnings)
	}
}

func TestAWSStatusPlain(t *testing.T) {
	summary := model.ToolSummary{
		ID:              "aws",
		DisplayName:     "AWS CLI",
		Category:        "cloud",
		Installed:       true,
		ConfiguredState: model.ConfiguredUnknown,
		Capabilities:    model.Capability{HasContexts: true},
		Warnings:        []string{"profile mismatch"},
		Errors:          []string{"region lookup failed"},
	}
	stubToolEvaluation(t, "aws", summary)

	opts := &rootOptions{Timeout: time.Second}
	stdout, stderr, err := executeTestCommand(t, newAWSCommand(opts, cliFakeRunner{}), "status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, needle := range []string{"TOOL", "aws", "Warnings:", "- aws: profile mismatch", "Errors:", "- aws: region lookup failed"} {
		if !strings.Contains(stdout, needle) {
			t.Fatalf("expected %q in output, got:\n%s", needle, stdout)
		}
	}
}

func TestAWSCurrentJSON(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"aws configure get region --profile prod": {
				Stdout: "us-west-2\n",
			},
			"aws configure get output --profile prod": {
				Stdout: "yaml\n",
			},
		},
	}

	t.Setenv("AWS_PROFILE", "prod")
	t.Setenv("AWS_REGION", "")
	t.Setenv("AWS_DEFAULT_REGION", "")
	t.Setenv("AWS_DEFAULT_OUTPUT", "")

	stdout, stderr, err := executeTestCommand(t, newAWSCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var got struct {
		Current  map[string]string `json:"current"`
		Warnings []string          `json:"warnings"`
		Errors   []string          `json:"errors"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if got.Current["profile"] != "prod" || got.Current["region"] != "us-west-2" || got.Current["output"] != "yaml" {
		t.Fatalf("unexpected current payload: %#v", got)
	}
	if len(got.Warnings) != 0 || len(got.Errors) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", got)
	}
}

func TestAWSCurrentPlainPrintsDiagnostics(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"aws configure get region --profile prod": {
				ExitCode: 1,
				Err:      errors.New("exit status 1"),
				Stderr:   "missing region",
			},
			"aws configure get output --profile prod": {
				Stdout: "json\n",
			},
		},
	}

	t.Setenv("AWS_PROFILE", "prod")
	t.Setenv("AWS_REGION", "")
	t.Setenv("AWS_DEFAULT_REGION", "")
	t.Setenv("AWS_DEFAULT_OUTPUT", "")

	stdout, stderr, err := executeTestCommand(t, newAWSCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, needle := range []string{"profile: prod", "output: json"} {
		if !strings.Contains(stdout, needle) {
			t.Fatalf("expected %q in stdout, got:\n%s", needle, stdout)
		}
	}
	if strings.Contains(stdout, "region:") {
		t.Fatalf("did not expect region line after failed lookup, got:\n%s", stdout)
	}
	if !strings.Contains(stderr, "error: missing region") {
		t.Fatalf("expected stderr diagnostics, got %q", stderr)
	}
}

func TestAWSListJSON(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"aws configure list-profiles": {
				Stdout: "prod\ndev\n",
			},
			"aws configure get region --profile prod": {
				Stdout: "us-west-2\n",
			},
			"aws configure get output --profile prod": {
				Stdout: "yaml\n",
			},
		},
	}

	t.Setenv("AWS_PROFILE", "prod")
	t.Setenv("AWS_REGION", "")
	t.Setenv("AWS_DEFAULT_REGION", "")
	t.Setenv("AWS_DEFAULT_OUTPUT", "")

	stdout, stderr, err := executeTestCommand(t, newAWSCommand(opts, runner), "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var got struct {
		Current  map[string]string `json:"current"`
		Profiles []struct {
			Name      string `json:"name"`
			IsCurrent bool   `json:"is_current"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if got.Current["profile"] != "prod" {
		t.Fatalf("unexpected current payload: %#v", got.Current)
	}
	if len(got.Profiles) != 2 || !got.Profiles[0].IsCurrent || got.Profiles[0].Name != "prod" {
		t.Fatalf("unexpected profiles payload: %#v", got.Profiles)
	}
}

func TestAWSListPlainMarksCurrentProfile(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"aws configure list-profiles": {
				Stdout: "prod\ndev\n",
			},
			"aws configure get region --profile prod": {
				Stdout: "us-west-2\n",
			},
			"aws configure get output --profile prod": {
				Stdout: "json\n",
			},
		},
	}

	t.Setenv("AWS_PROFILE", "prod")
	t.Setenv("AWS_REGION", "")
	t.Setenv("AWS_DEFAULT_REGION", "")
	t.Setenv("AWS_DEFAULT_OUTPUT", "")

	stdout, stderr, err := executeTestCommand(t, newAWSCommand(opts, runner), "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, needle := range []string{"* prod", "  dev"} {
		if !strings.Contains(stdout, needle) {
			t.Fatalf("expected %q in stdout, got:\n%s", needle, stdout)
		}
	}
}

func TestAWSListPlainReportsListFailure(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"aws configure list-profiles": {
				ExitCode: 1,
				Err:      errors.New("exit status 1"),
				Stderr:   "profiles unavailable",
			},
		},
	}

	t.Setenv("AWS_PROFILE", "prod")
	t.Setenv("AWS_REGION", "us-west-2")
	t.Setenv("AWS_DEFAULT_REGION", "")
	t.Setenv("AWS_DEFAULT_OUTPUT", "json")

	stdout, stderr, err := executeTestCommand(t, newAWSCommand(opts, runner), "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	for _, needle := range []string{
		"error: profiles unavailable",
		"error: aws configure list-profiles failed (exit=1)",
	} {
		if !strings.Contains(stderr, needle) {
			t.Fatalf("expected %q in stderr, got %q", needle, stderr)
		}
	}
}

func TestAWSListJSONReportsListFailure(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"aws configure list-profiles": {
				ExitCode: 1,
				Err:      errors.New("exit status 1"),
				Stderr:   "profiles unavailable",
			},
		},
	}

	t.Setenv("AWS_PROFILE", "prod")
	t.Setenv("AWS_REGION", "us-west-2")
	t.Setenv("AWS_DEFAULT_REGION", "")
	t.Setenv("AWS_DEFAULT_OUTPUT", "json")

	stdout, _, err := executeTestCommand(t, newAWSCommand(opts, runner), "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got struct {
		Errors []string `json:"errors"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if len(got.Errors) != 2 || got.Errors[0] != "profiles unavailable" {
		t.Fatalf("unexpected errors payload: %#v", got)
	}
}

func TestRootCommandIncludesAWS(t *testing.T) {
	cmd := NewRootCommand()

	for _, child := range cmd.Commands() {
		if child.Name() == "aws" {
			return
		}
	}
	t.Fatal("expected root command to register aws")
}
