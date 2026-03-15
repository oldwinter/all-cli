package cli

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
)

func TestAliyunStatusJSON(t *testing.T) {
	summary := model.ToolSummary{
		ID:              "aliyun",
		DisplayName:     "Aliyun CLI",
		Category:        "cloud",
		Installed:       true,
		InstallPath:     "/usr/bin/aliyun",
		ConfiguredState: model.ConfiguredYes,
		Configured:      true,
		Capabilities:    model.Capability{HasContexts: true},
		Current: map[string]string{
			"profile": "default",
			"region":  "cn-hangzhou",
		},
	}
	stubToolEvaluation(t, "aliyun", summary)

	opts := &rootOptions{JSON: true, Timeout: time.Second}
	stdout, stderr, err := executeTestCommand(t, newAliyunCommand(opts, cliFakeRunner{}), "status")
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
	if got.ID != "aliyun" || got.Current["profile"] != "default" {
		t.Fatalf("unexpected summary: %#v", got)
	}
}

func TestAliyunStatusPlain(t *testing.T) {
	summary := model.ToolSummary{
		ID:              "aliyun",
		DisplayName:     "Aliyun CLI",
		Category:        "cloud",
		Installed:       true,
		ConfiguredState: model.ConfiguredUnknown,
		Capabilities:    model.Capability{HasContexts: true},
		Warnings:        []string{"profile fallback"},
	}
	stubToolEvaluation(t, "aliyun", summary)

	opts := &rootOptions{Timeout: time.Second}
	stdout, stderr, err := executeTestCommand(t, newAliyunCommand(opts, cliFakeRunner{}), "status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, needle := range []string{"TOOL", "aliyun", "Warnings:", "- aliyun: profile fallback"} {
		if !strings.Contains(stdout, needle) {
			t.Fatalf("expected %q in output, got:\n%s", needle, stdout)
		}
	}
}

func TestAliyunCurrentJSON(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"aliyun configure list": {
				Stdout: `Profile   | Credential         | Valid   | Region      | Language
--------- | ------------------ | ------- | ----------- | --------
default * | AK:***6ps          | Valid   | cn-hangzhou | zh
`,
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newAliyunCommand(opts, runner), "current")
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
	if got.Current["profile"] != "default" || got.Current["language"] != "zh" || got.Current["valid"] != "Valid" {
		t.Fatalf("unexpected current payload: %#v", got.Current)
	}
}

func TestAliyunCurrentPlainPrintsFields(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"aliyun configure list": {
				Stdout: `Profile   | Credential         | Valid   | Region      | Language
--------- | ------------------ | ------- | ----------- | --------
default * | AK:***6ps          | Valid   | cn-hangzhou | zh
`,
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newAliyunCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, needle := range []string{"profile: default", "region: cn-hangzhou", "language: zh", "valid: Valid"} {
		if !strings.Contains(stdout, needle) {
			t.Fatalf("expected %q in stdout, got:\n%s", needle, stdout)
		}
	}
}

func TestAliyunCurrentPlainReportsErrors(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"aliyun configure list": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "config unreadable",
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newAliyunCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	for _, needle := range []string{
		"error: config unreadable",
		"error: aliyun configure list failed (exit=1)",
	} {
		if !strings.Contains(stderr, needle) {
			t.Fatalf("expected %q in stderr, got %q", needle, stderr)
		}
	}
}

func TestAliyunCurrentJSONReportsErrors(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"aliyun configure list": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "config unreadable",
			},
		},
	}

	stdout, _, err := executeTestCommand(t, newAliyunCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got struct {
		Errors []string `json:"errors"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if len(got.Errors) != 2 || got.Errors[0] != "config unreadable" {
		t.Fatalf("unexpected errors payload: %#v", got)
	}
}

func TestAliyunListJSON(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"aliyun configure list": {
				Stdout: `Profile   | Credential         | Valid   | Region      | Language
--------- | ------------------ | ------- | ----------- | --------
default * | AK:***6ps          | Valid   | cn-hangzhou | zh
dev       | AK:***123          | Invalid | us-east-1   | en
`,
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newAliyunCommand(opts, runner), "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var got struct {
		Profiles []struct {
			Name      string `json:"name"`
			IsCurrent bool   `json:"is_current"`
			Region    string `json:"region"`
			Language  string `json:"language"`
			Valid     string `json:"valid"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if len(got.Profiles) != 2 || !got.Profiles[0].IsCurrent || got.Profiles[1].Name != "dev" {
		t.Fatalf("unexpected profiles payload: %#v", got.Profiles)
	}
}

func TestAliyunListPlainMarksCurrentProfile(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"aliyun configure list": {
				Stdout: `Profile   | Credential         | Valid   | Region      | Language
--------- | ------------------ | ------- | ----------- | --------
default * | AK:***6ps          | Valid   | cn-hangzhou | zh
dev       | AK:***123          | Invalid | us-east-1   | en
`,
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newAliyunCommand(opts, runner), "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, needle := range []string{"* default", "region=cn-hangzhou", "language=zh", "valid=Valid", "  dev"} {
		if !strings.Contains(stdout, needle) {
			t.Fatalf("expected %q in stdout, got:\n%s", needle, stdout)
		}
	}
}

func TestAliyunListPlainReportsErrors(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"aliyun configure list": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "config unreadable",
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newAliyunCommand(opts, runner), "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	for _, needle := range []string{
		"error: config unreadable",
		"error: aliyun configure list failed (exit=1)",
	} {
		if !strings.Contains(stderr, needle) {
			t.Fatalf("expected %q in stderr, got %q", needle, stderr)
		}
	}
}

func TestAliyunListJSONReportsErrors(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"aliyun configure list": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "config unreadable",
			},
		},
	}

	stdout, _, err := executeTestCommand(t, newAliyunCommand(opts, runner), "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got struct {
		Errors []string `json:"errors"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if len(got.Errors) != 2 || got.Errors[0] != "config unreadable" {
		t.Fatalf("unexpected errors payload: %#v", got)
	}
}

func TestRootCommandIncludesAliyun(t *testing.T) {
	cmd := NewRootCommand()

	for _, child := range cmd.Commands() {
		if child.Name() == "aliyun" {
			return
		}
	}
	t.Fatal("expected root command to register aliyun")
}
