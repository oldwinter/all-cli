package diagnose

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/oldwinter/all-cli/internal/model"
)

const (
	ProfileAgent = "agent"
	ProfileHuman = "human"
	ProfileCI    = "ci"
)

type Options struct {
	Profile string
}

type FixOptions struct {
	DryRun bool
}

func Generate(status model.StatusReport, opts Options) model.DiagnosticReport {
	profile := normalizeProfile(opts.Profile)
	sourceSchema := strings.TrimSpace(status.SchemaVersion)
	if sourceSchema == "" {
		sourceSchema = model.SchemaVersionV01
	}

	report := model.DiagnosticReport{
		SchemaVersion:       model.DiagnosticSchemaVersionV01,
		GeneratedAt:         time.Now(),
		SourceSchemaVersion: sourceSchema,
		Profile:             profile,
		Tools:               append([]model.ToolSummary(nil), status.Tools...),
		Diagnostics:         []model.DiagnosticItem{},
	}
	if !status.GeneratedAt.IsZero() {
		report.GeneratedAt = status.GeneratedAt
	}

	for _, tool := range status.Tools {
		report.Diagnostics = append(report.Diagnostics, diagnosticsForTool(tool)...)
	}
	report.Summary = summarizeDiagnostics(report.Diagnostics)
	return report
}

func BuildFixPlan(report model.DiagnosticReport, opts FixOptions) model.FixPlan {
	plan := model.FixPlan{
		SchemaVersion: model.FixPlanSchemaVersionV01,
		GeneratedAt:   time.Now(),
		DryRun:        opts.DryRun,
		Items:         make([]model.FixPlanItem, 0, len(report.Diagnostics)),
	}
	if !report.GeneratedAt.IsZero() {
		plan.GeneratedAt = report.GeneratedAt
	}

	for _, diagnostic := range report.Diagnostics {
		action := firstAction(diagnostic)
		supported := diagnostic.SafeToAutofix && action.ID != "" && !action.Mutates
		item := model.FixPlanItem{
			DiagnosticID: diagnostic.ID,
			RelatedTool:  diagnostic.RelatedTool,
			Action:       action,
			Supported:    supported,
			WillRun:      false,
			Mutates:      action.Mutates,
		}
		if supported {
			item.Reason = "automatic execution is disabled in dry-run mode"
		} else {
			item.Reason = "automatic mutation is not allowlisted; use the suggested manual action"
		}
		plan.Items = append(plan.Items, item)
	}

	plan.Summary.Total = len(plan.Items)
	for _, item := range plan.Items {
		if item.Supported {
			plan.Summary.Supported++
		} else {
			plan.Summary.Blocked++
		}
	}
	return plan
}

func DiffSnapshots(before, after model.StatusReport) model.SnapshotDiffReport {
	report := model.SnapshotDiffReport{
		SchemaVersion: model.SnapshotDiffSchemaVersionV01,
		GeneratedAt:   time.Now(),
		Changes:       []model.SnapshotToolChange{},
	}

	beforeByID := indexTools(before.Tools)
	afterByID := indexTools(after.Tools)
	ids := make([]string, 0, len(beforeByID)+len(afterByID))
	seen := map[string]bool{}
	for id := range beforeByID {
		ids = append(ids, id)
		seen[id] = true
	}
	for id := range afterByID {
		if !seen[id] {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)

	for _, id := range ids {
		beforeTool, hadBefore := beforeByID[id]
		afterTool, hasAfter := afterByID[id]
		switch {
		case !hadBefore && hasAfter:
			tool := afterTool
			report.Changes = append(report.Changes, model.SnapshotToolChange{
				ToolID:     id,
				ChangeType: model.SnapshotChangeAdded,
				After:      &tool,
			})
			report.Summary.Added++
		case hadBefore && !hasAfter:
			tool := beforeTool
			report.Changes = append(report.Changes, model.SnapshotToolChange{
				ToolID:     id,
				ChangeType: model.SnapshotChangeRemoved,
				Before:     &tool,
			})
			report.Summary.Removed++
		default:
			fields := changedFields(beforeTool, afterTool)
			if len(fields) == 0 {
				continue
			}
			beforeCopy := beforeTool
			afterCopy := afterTool
			report.Changes = append(report.Changes, model.SnapshotToolChange{
				ToolID:     id,
				ChangeType: model.SnapshotChangeChanged,
				Fields:     fields,
				Before:     &beforeCopy,
				After:      &afterCopy,
			})
			report.Summary.Changed++
		}
	}

	return report
}

func diagnosticsForTool(tool model.ToolSummary) []model.DiagnosticItem {
	var out []model.DiagnosticItem
	if !tool.Installed {
		out = append(out, model.DiagnosticItem{
			ID:            tool.ID + ".not_installed",
			Severity:      model.DiagnosticInfo,
			Problem:       fmt.Sprintf("%s is not installed.", displayName(tool)),
			Evidence:      []string{fmt.Sprintf("binary for %s was not found in PATH", tool.ID)},
			RelatedTool:   tool.ID,
			SafeToAutofix: false,
			SuggestedActions: []model.SuggestedAction{{
				ID:          "install_tool",
				Title:       "Install " + displayName(tool),
				Description: "Install the CLI binary, then rerun all-cli status or all-cli diagnose.",
				Kind:        "install",
				Mutates:     true,
			}},
		})
		return out
	}

	if len(tool.Errors) > 0 {
		out = append(out, model.DiagnosticItem{
			ID:            tool.ID + ".collection_errors",
			Severity:      model.DiagnosticError,
			Problem:       fmt.Sprintf("%s status collection reported errors.", displayName(tool)),
			Evidence:      appendPrefixed("error: ", tool.Errors),
			RelatedTool:   tool.ID,
			SafeToAutofix: false,
			SuggestedActions: []model.SuggestedAction{{
				ID:          "rerun_with_timeout",
				Title:       "Rerun with a longer timeout",
				Description: "Inspect whether the underlying CLI is slow, blocked on login, or returning a non-zero exit.",
				Kind:        "inspect",
				Command:     []string{"all-cli", "status", "--tools", tool.ID, "--timeout", "15s"},
				Mutates:     false,
			}},
		})
	}

	if len(tool.Warnings) > 0 {
		out = append(out, model.DiagnosticItem{
			ID:            tool.ID + ".warnings",
			Severity:      model.DiagnosticWarning,
			Problem:       fmt.Sprintf("%s status collection reported warnings.", displayName(tool)),
			Evidence:      appendPrefixed("warning: ", tool.Warnings),
			RelatedTool:   tool.ID,
			SafeToAutofix: false,
			SuggestedActions: []model.SuggestedAction{{
				ID:          "inspect_warning",
				Title:       "Inspect warning",
				Description: "Read the warning and decide whether this is expected for the current environment.",
				Kind:        "inspect",
				Command:     []string{"all-cli", "status", "--tools", tool.ID},
				Mutates:     false,
			}},
		})
	}

	switch tool.ConfiguredState {
	case model.ConfiguredNo:
		out = append(out, model.DiagnosticItem{
			ID:            tool.ID + ".configured_state",
			Severity:      model.DiagnosticWarning,
			Problem:       fmt.Sprintf("%s is installed but not configured.", displayName(tool)),
			Evidence:      configuredEvidence(tool),
			RelatedTool:   tool.ID,
			SafeToAutofix: false,
			SuggestedActions: []model.SuggestedAction{{
				ID:          "configure_tool",
				Title:       "Configure " + displayName(tool),
				Description: configureDescription(tool),
				Kind:        "configure",
				Command:     []string{"all-cli", tool.ID, "status"},
				Mutates:     false,
			}},
		})
	case model.ConfiguredUnknown:
		if len(tool.Errors) == 0 {
			out = append(out, model.DiagnosticItem{
				ID:            tool.ID + ".configured_state_unknown",
				Severity:      model.DiagnosticWarning,
				Problem:       fmt.Sprintf("%s configuration state is unknown.", displayName(tool)),
				Evidence:      configuredEvidence(tool),
				RelatedTool:   tool.ID,
				SafeToAutofix: false,
				SuggestedActions: []model.SuggestedAction{{
					ID:          "inspect_status",
					Title:       "Inspect status",
					Description: "Run the focused status command and inspect any tool-specific diagnostics.",
					Kind:        "inspect",
					Command:     []string{"all-cli", "status", "--tools", tool.ID},
					Mutates:     false,
				}},
			})
		}
	}

	if tool.Capabilities.HasContexts && tool.Configured && len(tool.Current) == 0 {
		out = append(out, model.DiagnosticItem{
			ID:            tool.ID + ".missing_current_context",
			Severity:      model.DiagnosticWarning,
			Problem:       fmt.Sprintf("%s is configured but has no current context snapshot.", displayName(tool)),
			Evidence:      []string{"capabilities.has_contexts=true", "current is empty"},
			RelatedTool:   tool.ID,
			SafeToAutofix: false,
			SuggestedActions: []model.SuggestedAction{{
				ID:          "list_contexts",
				Title:       "List contexts",
				Description: "List available contexts and choose the intended target before operating.",
				Kind:        "inspect",
				Command:     []string{"all-cli", tool.ID, "list"},
				Mutates:     false,
			}},
		})
	}

	return out
}

func summarizeDiagnostics(items []model.DiagnosticItem) model.DiagnosticSummary {
	summary := model.DiagnosticSummary{Total: len(items)}
	for _, item := range items {
		switch item.Severity {
		case model.DiagnosticInfo:
			summary.Info++
		case model.DiagnosticWarning:
			summary.Warning++
		case model.DiagnosticError:
			summary.Error++
		}
	}
	return summary
}

func normalizeProfile(profile string) string {
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case ProfileHuman:
		return ProfileHuman
	case ProfileCI:
		return ProfileCI
	default:
		return ProfileAgent
	}
}

func displayName(tool model.ToolSummary) string {
	if strings.TrimSpace(tool.DisplayName) != "" {
		return strings.TrimSpace(tool.DisplayName)
	}
	return tool.ID
}

func appendPrefixed(prefix string, messages []string) []string {
	out := make([]string, 0, len(messages))
	for _, msg := range messages {
		msg = strings.TrimSpace(msg)
		if msg == "" {
			continue
		}
		out = append(out, prefix+msg)
	}
	return out
}

func configuredEvidence(tool model.ToolSummary) []string {
	evidence := []string{"configured_state=" + string(tool.ConfiguredState)}
	if strings.TrimSpace(tool.Metadata.ConfiguredWhen) != "" {
		evidence = append(evidence, "configured_when: "+strings.TrimSpace(tool.Metadata.ConfiguredWhen))
	}
	return evidence
}

func configureDescription(tool model.ToolSummary) string {
	if strings.TrimSpace(tool.Metadata.ConfiguredWhen) != "" {
		return "Expected configured condition: " + strings.TrimSpace(tool.Metadata.ConfiguredWhen)
	}
	return "Authenticate or create local configuration for this CLI, then rerun diagnostics."
}

func firstAction(diagnostic model.DiagnosticItem) model.SuggestedAction {
	if len(diagnostic.SuggestedActions) == 0 {
		return model.SuggestedAction{
			ID:          "inspect_diagnostic",
			Title:       "Inspect diagnostic",
			Description: "Read the diagnostic evidence and decide the appropriate manual fix.",
			Kind:        "inspect",
			Mutates:     false,
		}
	}
	return diagnostic.SuggestedActions[0]
}

func indexTools(tools []model.ToolSummary) map[string]model.ToolSummary {
	out := make(map[string]model.ToolSummary, len(tools))
	for _, tool := range tools {
		id := strings.TrimSpace(tool.ID)
		if id == "" {
			continue
		}
		out[id] = tool
	}
	return out
}

func changedFields(before, after model.ToolSummary) []string {
	var fields []string
	if before.DisplayName != after.DisplayName {
		fields = append(fields, "display_name")
	}
	if before.Category != after.Category {
		fields = append(fields, "category")
	}
	if before.Installed != after.Installed {
		fields = append(fields, "installed")
	}
	if before.InstallPath != after.InstallPath {
		fields = append(fields, "install_path")
	}
	if before.ConfiguredState != after.ConfiguredState {
		fields = append(fields, "configured_state")
	}
	if before.Configured != after.Configured {
		fields = append(fields, "configured")
	}
	if !reflect.DeepEqual(before.Capabilities, after.Capabilities) {
		fields = append(fields, "capabilities")
	}
	if !reflect.DeepEqual(before.Current, after.Current) {
		fields = append(fields, "current")
	}
	if !reflect.DeepEqual(before.Warnings, after.Warnings) {
		fields = append(fields, "warnings")
	}
	if !reflect.DeepEqual(before.Errors, after.Errors) {
		fields = append(fields, "errors")
	}
	return fields
}
