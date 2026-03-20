package tools

import (
	"context"
	"strings"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
)

// Evaluate checks whether a tool is installed and configured, returning a ToolSummary.
func Evaluate(ctx context.Context, def ToolDefinition, runner execx.Runner) model.ToolSummary {
	installPath, err := execx.LookPath(def.Binary)
	installed := err == nil

	summary := model.ToolSummary{
		ID:          def.ID,
		DisplayName: def.DisplayName,
		Category:    def.Category,
		Installed:   installed,
		Capabilities: model.Capability{
			HasContexts: def.Capabilities.HasContexts,
			CanSwitch:   def.Capabilities.CanSwitch,
		},
		ConfiguredState: model.ConfiguredUnknown,
		Configured:      false,
		Metadata:        MetadataForTool(def.ID),
	}
	if installed {
		summary.InstallPath = installPath
	}

	if def.ConfigCheck != nil {
		state, warnings, errs := def.ConfigCheck(ctx, runner, installed)
		summary.ConfiguredState = state
		summary.Configured = state == model.ConfiguredYes || state == model.ConfiguredNA
		summary.Warnings = append(summary.Warnings, warnings...)
		summary.Errors = append(summary.Errors, errs...)
	}

	if def.Current != nil {
		cur, warnings, errs := def.Current(ctx, runner, installed)
		if len(cur) > 0 {
			summary.Current = cur
		}
		summary.Warnings = append(summary.Warnings, warnings...)
		summary.Errors = append(summary.Errors, errs...)
	}

	summary.Warnings = dedupeStrings(summary.Warnings)
	summary.Errors = dedupeStrings(summary.Errors)

	return summary
}

func dedupeStrings(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}
