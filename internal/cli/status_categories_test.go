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

func TestStatusCommandFiltersToolsByCategories(t *testing.T) {
	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "aws", DisplayName: "AWS CLI", Category: "cloud", Binary: "aws"},
		{ID: "kubectl", DisplayName: "kubectl", Category: "k8s", Binary: "kubectl"},
		{ID: "gh", DisplayName: "gh", Category: "code", Binary: "gh"},
	})
	oldEvaluate := evaluateToolSummary
	evaluateToolSummary = func(_ context.Context, def tools.ToolDefinition, _ execx.Runner) model.ToolSummary {
		return model.ToolSummary{ID: def.ID, Category: def.Category, Installed: true}
	}
	t.Cleanup(func() { evaluateToolSummary = oldEvaluate })
	stubShowStatusSpinner(t, false)

	opts := &rootOptions{JSON: true, Timeout: time.Second}
	stdout, _, err := executeTestCommand(t, newStatusCommand(opts, cliFakeRunner{}),
		"--tools", "aws,kubectl", "--categories", "cloud,code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got model.StatusReport
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if len(got.Tools) != 1 || got.Tools[0].ID != "aws" {
		t.Fatalf("unexpected category-filtered tools: %#v", got.Tools)
	}
}

func TestStatusCommandRejectsUnknownCategories(t *testing.T) {
	stubStatusRegistry(t, []tools.ToolDefinition{
		{ID: "aws", DisplayName: "AWS CLI", Category: "cloud", Binary: "aws"},
	})
	stubShowStatusSpinner(t, false)

	opts := &rootOptions{JSON: true, Timeout: time.Second}
	_, _, err := executeTestCommand(t, newStatusCommand(opts, cliFakeRunner{}), "--categories", "cloud,workflows")
	if err == nil || !strings.Contains(err.Error(), "unknown categories: workflows") {
		t.Fatalf("unexpected error: %v", err)
	}
}
