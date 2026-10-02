package diagnose

import (
	"errors"
	"testing"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
)

func diagnosticIDs(items []model.DiagnosticItem) map[string]model.DiagnosticItem {
	out := make(map[string]model.DiagnosticItem, len(items))
	for _, item := range items {
		out[item.ID] = item
	}
	return out
}

func TestDiagnosticsForToolMatrix(t *testing.T) {
	tests := []struct {
		name    string
		tool    model.ToolSummary
		wantIDs []string
	}{
		{
			name: "healthy",
			tool: model.ToolSummary{
				ID: "kubectl", Installed: true, Configured: true,
				ConfiguredState: model.ConfiguredYes,
				Current:         map[string]string{"context": "orb"},
			},
			wantIDs: nil,
		},
		{
			name:    "not installed",
			tool:    model.ToolSummary{ID: "helm", Installed: false},
			wantIDs: []string{"helm.not_installed"},
		},
		{
			name: "collection errors suppress unknown-config diagnostic",
			tool: model.ToolSummary{
				ID: "aws", Installed: true,
				ConfiguredState: model.ConfiguredUnknown,
				Errors:          []string{"configure list failed"},
			},
			wantIDs: []string{"aws.collection_errors"},
		},
		{
			name: "warnings",
			tool: model.ToolSummary{
				ID: "kubectl", Installed: true, Configured: true,
				ConfiguredState: model.ConfiguredYes,
				Current:         map[string]string{"context": "orb"},
				Warnings:        []string{"namespace unset"},
			},
			wantIDs: []string{"kubectl.warnings"},
		},
		{
			name: "configured no",
			tool: model.ToolSummary{
				ID: "gh", Installed: true,
				ConfiguredState: model.ConfiguredNo,
			},
			wantIDs: []string{"gh.configured_state"},
		},
		{
			name: "configured unknown without errors",
			tool: model.ToolSummary{
				ID: "glab", Installed: true,
				ConfiguredState: model.ConfiguredUnknown,
			},
			wantIDs: []string{"glab.configured_state_unknown"},
		},
		{
			name: "missing current context",
			tool: model.ToolSummary{
				ID: "kubectl", Installed: true, Configured: true,
				ConfiguredState: model.ConfiguredYes,
				Capabilities:    model.Capability{HasContexts: true},
			},
			wantIDs: []string{"kubectl.missing_current_context"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := diagnosticIDs(diagnosticsForTool(tc.tool))
			if len(got) != len(tc.wantIDs) {
				t.Fatalf("diagnostics = %#v, want %v", got, tc.wantIDs)
			}
			for _, id := range tc.wantIDs {
				if _, ok := got[id]; !ok {
					t.Fatalf("missing diagnostic %s in %#v", id, got)
				}
			}
		})
	}
}

func TestDiagnosticsForToolGHSuggestsUpgrade(t *testing.T) {
	tool := model.ToolSummary{
		ID: "gh", Installed: true,
		ConfiguredState: model.ConfiguredUnknown,
		Errors:          []string{"gh: unknown flag: --json"},
	}
	items := diagnosticsForTool(tool)
	if len(items) != 1 {
		t.Fatalf("items = %#v", items)
	}
	action := firstAction(items[0])
	if action.ID != "upgrade_or_check_gh_auth" {
		t.Fatalf("action = %#v, want gh upgrade hint", action)
	}
}

func TestDiagnosticsForToolConfigParseErrorSuggestsInspection(t *testing.T) {
	for _, tool := range []model.ToolSummary{
		{
			ID: "kubectl", Installed: true,
			ConfiguredState: model.ConfiguredUnknown,
			Errors:          []string{`error loading config file "/home/u/.kube/config": yaml: did not find expected ',' or ']'`},
		},
		{
			ID: "gh", Installed: true,
			ConfiguredState: model.ConfiguredUnknown,
			Errors:          []string{"invalid config file /home/u/.config/gh/hosts.yml: invalid yaml"},
		},
	} {
		items := diagnosticsForTool(tool)
		action := firstAction(items[0])
		if action.ID != "inspect_config_file" {
			t.Fatalf("%s: action = %#v, want inspect_config_file", tool.ID, action)
		}
		if action.Mutates {
			t.Fatalf("%s: inspect action must not mutate: %#v", tool.ID, action)
		}
	}
}

func TestDiagnosticsForToolGenericErrorsSuggestTimeout(t *testing.T) {
	tool := model.ToolSummary{
		ID: "aws", Installed: true,
		ConfiguredState: model.ConfiguredUnknown,
		Errors:          []string{"context deadline exceeded"},
	}
	items := diagnosticsForTool(tool)
	if action := firstAction(items[0]); action.ID != "rerun_with_timeout" {
		t.Fatalf("action = %#v, want rerun_with_timeout", action)
	}
}

func TestDoctorCommandFailureComposesParts(t *testing.T) {
	tests := []struct {
		name string
		res  execx.CmdResult
		want string
	}{
		{
			name: "err and stderr",
			res:  execx.CmdResult{Err: errors.New("exit status 1"), Stderr: "boom", ExitCode: 1},
			want: "exit status 1: boom",
		},
		{
			name: "stderr only",
			res:  execx.CmdResult{Stderr: "bad flag", ExitCode: 2},
			want: "bad flag",
		},
		{
			name: "stdout fallback",
			res:  execx.CmdResult{Stdout: "usage text", ExitCode: 2},
			want: "usage text",
		},
		{
			name: "exit code only",
			res:  execx.CmdResult{ExitCode: 7},
			want: "command exited with code 7",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := doctorCommandFailure(tc.res); got != tc.want {
				t.Fatalf("doctorCommandFailure = %q, want %q", got, tc.want)
			}
		})
	}
}
