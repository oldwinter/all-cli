package cli

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
)

func TestK9sStatusJSON(t *testing.T) {
	summary := model.ToolSummary{
		ID:              "k9s",
		DisplayName:     "k9s",
		Category:        "tui",
		Installed:       true,
		InstallPath:     "/usr/bin/k9s",
		ConfiguredState: model.ConfiguredNA,
		Configured:      true,
		Capabilities:    model.Capability{HasContexts: true},
		Current:         map[string]string{"context": "prod-cluster"},
	}
	stubToolEvaluation(t, "k9s", summary)

	opts := &rootOptions{JSON: true, Timeout: time.Second}
	stdout, stderr, err := executeTestCommand(t, newK9sCommand(opts, cliFakeRunner{}), "status")
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
	if got.ID != "k9s" || got.Current["context"] != "prod-cluster" {
		t.Fatalf("unexpected summary: %#v", got)
	}
}

func TestK9sStatusPlain(t *testing.T) {
	summary := model.ToolSummary{
		ID:              "k9s",
		DisplayName:     "k9s",
		Category:        "tui",
		Installed:       true,
		ConfiguredState: model.ConfiguredNA,
		Configured:      true,
		Capabilities:    model.Capability{HasContexts: true},
		Warnings:        []string{"namespace not set"},
	}
	stubToolEvaluation(t, "k9s", summary)

	opts := &rootOptions{Timeout: time.Second}
	stdout, stderr, err := executeTestCommand(t, newK9sCommand(opts, cliFakeRunner{}), "status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, needle := range []string{"TOOL", "k9s", "Warnings:", "- k9s: namespace not set"} {
		if !strings.Contains(stdout, needle) {
			t.Fatalf("expected %q in output, got:\n%s", needle, stdout)
		}
	}
}

func TestK9sCurrentJSON(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"kubectl config current-context": {
				Stdout: "prod-cluster\n",
			},
			"kubectl config view --minify --output jsonpath={..namespace}{\"\\n\"}": {
				Stdout: "payments\n",
			},
			"k9s info": {
				Stdout: "Config:  /tmp/k9s.yaml\n",
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newK9sCommand(opts, runner), "current")
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
	if got.Current["context"] != "prod-cluster" || got.Current["namespace"] != "payments" || got.Current["config"] != "/tmp/k9s.yaml" {
		t.Fatalf("unexpected current payload: %#v", got.Current)
	}
}

func TestK9sCurrentPlainPrintsDiagnostics(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"kubectl config current-context": {
				Stdout: "prod-cluster\n",
			},
			"kubectl config view --minify --output jsonpath={..namespace}{\"\\n\"}": {
				Stdout: "",
			},
			"k9s info": {
				Stdout: "Config:  /tmp/k9s.yaml\n",
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newK9sCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, needle := range []string{"context: prod-cluster", "config: /tmp/k9s.yaml"} {
		if !strings.Contains(stdout, needle) {
			t.Fatalf("expected %q in stdout, got:\n%s", needle, stdout)
		}
	}
	if strings.Contains(stdout, "namespace:") {
		t.Fatalf("did not expect namespace line, got:\n%s", stdout)
	}
	if !strings.Contains(stderr, "warning: k9s context detected via kubeconfig; namespace not set in kubeconfig") {
		t.Fatalf("expected warning in stderr, got %q", stderr)
	}
	if strings.Contains(stdout, "no k9s context. Check: all-cli k9s status") {
		t.Fatalf("did not expect status next step when fields are present, got:\n%s", stdout)
	}
}

func TestK9sCurrentPlainEmptyPointsAtStatus(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"kubectl config current-context": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "error: current-context is not set",
			},
			"kubectl config view --minify --output jsonpath={..namespace}{\"\\n\"}": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "error: current-context is not set",
			},
			"k9s info": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "k9s: command not found",
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newK9sCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "no k9s context. Check: all-cli k9s status") {
		t.Fatalf("expected status next step in stdout, got:\n%s", stdout)
	}
	if strings.Contains(stdout, "context:") || strings.Contains(stdout, "namespace:") || strings.Contains(stdout, "config:") {
		t.Fatalf("did not expect context/namespace/config lines, got:\n%s", stdout)
	}
	if !strings.Contains(stderr, "error: kubectl config current-context failed") {
		t.Fatalf("expected current-context error on stderr, got %q", stderr)
	}
}

func TestRootCommandIncludesK9s(t *testing.T) {
	cmd := NewRootCommand()
	for _, child := range cmd.Commands() {
		if child.Name() == "k9s" {
			return
		}
	}
	t.Fatal("expected root command to register k9s")
}
