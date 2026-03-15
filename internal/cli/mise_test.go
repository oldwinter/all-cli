package cli

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
)

func TestMiseStatusJSON(t *testing.T) {
	summary := model.ToolSummary{
		ID:              "mise",
		DisplayName:     "Mise",
		Category:        "env",
		Installed:       true,
		InstallPath:     "/usr/bin/mise",
		ConfiguredState: model.ConfiguredNA,
		Configured:      true,
		Capabilities:    model.Capability{HasContexts: true},
		Current:         map[string]string{"go": "1.26.1"},
	}
	stubToolEvaluation(t, "mise", summary)

	opts := &rootOptions{JSON: true, Timeout: time.Second}
	stdout, stderr, err := executeTestCommand(t, newMiseCommand(opts, cliFakeRunner{}), "status")
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
	if got.ID != "mise" || got.Current["go"] != "1.26.1" {
		t.Fatalf("unexpected summary: %#v", got)
	}
}

func TestMiseStatusPlain(t *testing.T) {
	summary := model.ToolSummary{
		ID:              "mise",
		DisplayName:     "Mise",
		Category:        "env",
		Installed:       true,
		ConfiguredState: model.ConfiguredNA,
		Configured:      true,
		Capabilities:    model.Capability{HasContexts: true},
	}
	stubToolEvaluation(t, "mise", summary)

	opts := &rootOptions{Timeout: time.Second}
	stdout, stderr, err := executeTestCommand(t, newMiseCommand(opts, cliFakeRunner{}), "status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, needle := range []string{"TOOL", "mise"} {
		if !strings.Contains(stdout, needle) {
			t.Fatalf("expected %q in output, got:\n%s", needle, stdout)
		}
	}
}

func TestMiseCurrentJSON(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"mise current": {
				Stdout: "node 25.8.1\ngo 1.26.1\n",
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newMiseCommand(opts, runner), "current")
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
	if got.Current["go"] != "1.26.1" || got.Current["node"] != "25.8.1" {
		t.Fatalf("unexpected current: %#v", got.Current)
	}
}

func TestMiseCurrentPlainSorted(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"mise current": {
				Stdout: "python 3.14.0\ngo 1.26.1\nnode 25.8.1\n",
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newMiseCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	want := []string{"go: 1.26.1", "node: 25.8.1", "python: 3.14.0"}
	if len(lines) != len(want) {
		t.Fatalf("unexpected output lines: %#v", lines)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Fatalf("line %d = %q, want %q", i, lines[i], want[i])
		}
	}
}

func TestMiseCurrentPlainWithWarning(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"mise current": {
				Stdout: "go 1.26.1\nbroken-line\n",
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newMiseCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "go: 1.26.1") {
		t.Fatalf("unexpected stdout:\n%s", stdout)
	}
	if !strings.Contains(stderr, "warning: unexpected mise current output line: broken-line") {
		t.Fatalf("expected warning in stderr, got %q", stderr)
	}
}

func TestMiseCurrentPlainError(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"mise current": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "mise unavailable",
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newMiseCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "error: mise unavailable") {
		t.Fatalf("expected stderr error, got %q", stderr)
	}
}

func TestRootCommandIncludesMise(t *testing.T) {
	cmd := NewRootCommand()
	for _, child := range cmd.Commands() {
		if child.Name() == "mise" {
			return
		}
	}
	t.Fatal("expected root command to register mise")
}
