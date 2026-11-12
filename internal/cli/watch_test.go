package cli

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
	"github.com/oldwinter/all-cli/internal/tools"
)

func TestWatchCommandRefreshesAndReportsChangedContexts(t *testing.T) {
	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "aws", Category: "cloud", Binary: "aws", Capabilities: model.Capability{HasContexts: true}},
		{ID: "fd", Category: "navigation", Binary: "fd"},
	})
	var calls atomic.Int32
	oldEvaluate := evaluateToolSummary
	evaluateToolSummary = func(_ context.Context, def tools.ToolDefinition, _ execx.Runner) model.ToolSummary {
		if def.ID != "aws" {
			t.Fatalf("unexpected tool evaluation: %s", def.ID)
		}
		profile := "dev"
		if calls.Add(1) > 1 {
			profile = "prod"
		}
		return model.ToolSummary{
			ID:           def.ID,
			Installed:    true,
			Capabilities: def.Capabilities,
			Current:      map[string]string{"profile": profile},
		}
	}
	t.Cleanup(func() { evaluateToolSummary = oldEvaluate })

	stdout, stderr, err := executeTestCommand(
		t,
		newWatchCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}),
		"--tools", "aws",
		"--interval", "100ms",
		"--frames", "2",
		"--no-clear",
	)
	if err != nil {
		t.Fatalf("watch: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if got := strings.Count(stdout, "all-cli watch"); got != 2 {
		t.Fatalf("expected two watch frames, got %d:\n%s", got, stdout)
	}
	if !strings.Contains(stdout, "profile=dev") || !strings.Contains(stdout, "profile=prod") {
		t.Fatalf("expected both context values, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Changed: aws\n") {
		t.Fatalf("expected changed context line, got:\n%s", stdout)
	}
	if strings.Contains(stdout, "fd") {
		t.Fatalf("expected non-context tool to be skipped, got:\n%s", stdout)
	}
}

func TestWatchCommandFirstFrameHasNoChangedLine(t *testing.T) {
	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "aws", Category: "cloud", Binary: "aws", Capabilities: model.Capability{HasContexts: true}},
	})
	oldEvaluate := evaluateToolSummary
	evaluateToolSummary = func(_ context.Context, def tools.ToolDefinition, _ execx.Runner) model.ToolSummary {
		return model.ToolSummary{
			ID:           def.ID,
			Installed:    true,
			Capabilities: def.Capabilities,
			Current:      map[string]string{"profile": "work"},
		}
	}
	t.Cleanup(func() { evaluateToolSummary = oldEvaluate })

	stdout, _, err := executeTestCommand(
		t,
		newWatchCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}),
		"--frames", "1",
		"--no-clear",
	)
	if err != nil {
		t.Fatalf("watch: %v", err)
	}
	if strings.Contains(stdout, "Changed:") {
		t.Fatalf("first frame should not report changes, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Current: 1 tools, 1 with a readable context\n") {
		t.Fatalf("expected current summary, got:\n%s", stdout)
	}
}

func TestWatchCommandRejectsJSONOutput(t *testing.T) {
	_, _, err := executeTestCommand(
		t,
		newWatchCommand(&rootOptions{JSON: true, Timeout: time.Second}, cliFakeRunner{}),
		"--frames", "1",
	)
	if err == nil || !strings.Contains(err.Error(), "all-cli current --json") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWatchCommandRejectsInvalidIntervalAndFrames(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "interval", args: []string{"--interval", "10ms"}, want: "must be at least 100ms"},
		{name: "frames", args: []string{"--frames", "-1"}, want: "must be zero or greater"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := executeTestCommand(
				t,
				newWatchCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}),
				tt.args...,
			)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestChangedContextToolIDs(t *testing.T) {
	previous := map[string]map[string]string{
		"aws":    {"profile": "dev"},
		"docker": {"context": "desktop-linux"},
		"gh":     {},
	}
	current := map[string]map[string]string{
		"aws":     {"profile": "prod"},
		"gh":      {},
		"kubectl": {"context": "staging"},
	}

	got := strings.Join(changedContextToolIDs(previous, current), ",")
	want := "aws,docker (removed),kubectl (new)"
	if got != want {
		t.Fatalf("changes = %q, want %q", got, want)
	}
}

func TestContextValuesByToolIgnoresBlankValues(t *testing.T) {
	values := contextValuesByTool([]model.ToolSummary{
		{ID: "aws", Current: map[string]string{"profile": " work ", "region": "  "}},
		{ID: "docker"},
	})
	if got := values["aws"]["profile"]; got != "work" {
		t.Fatalf("profile = %q, want work", got)
	}
	if _, exists := values["aws"]["region"]; exists {
		t.Fatal("blank current value should be omitted")
	}
	if len(values["docker"]) != 0 {
		t.Fatalf("missing current map should stay empty, got %#v", values["docker"])
	}
}

func TestRootRegistersWatchAsPrimaryCommand(t *testing.T) {
	root := NewRootCommand()
	watch, _, err := root.Find([]string{"watch"})
	if err != nil {
		t.Fatalf("find watch command: %v", err)
	}
	if watch.Name() != "watch" || watch.GroupID != "primary" {
		t.Fatalf("unexpected watch registration: name=%q group=%q", watch.Name(), watch.GroupID)
	}
}
