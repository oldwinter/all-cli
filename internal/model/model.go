// Package model defines shared types for tool status reporting.
package model

import "time"

// SchemaVersionV01 is the current JSON schema version for status output.
const SchemaVersionV01 = "v0.1"

// ConfiguredState represents the configuration status of a CLI tool.
type ConfiguredState string

const (
	ConfiguredYes     ConfiguredState = "yes"
	ConfiguredNo      ConfiguredState = "no"
	ConfiguredNA      ConfiguredState = "n/a"
	ConfiguredUnknown ConfiguredState = "unknown"
)

// Capability describes what context-like features a tool supports.
type Capability struct {
	HasContexts bool `json:"has_contexts"`
	CanSwitch   bool `json:"can_switch"`
}

// ToolMetadata provides human- and agent-readable context about a tool.
type ToolMetadata struct {
	Purpose                  string            `json:"purpose,omitempty"`
	ConfiguredWhen           string            `json:"configured_when,omitempty"`
	CurrentFieldDescriptions map[string]string `json:"current_field_descriptions,omitempty"`
	AgentActions             []string          `json:"agent_actions,omitempty"`
	Notes                    []string          `json:"notes,omitempty"`
}

type StatusLegend struct {
	Installed       string            `json:"installed,omitempty"`
	ConfiguredState map[string]string `json:"configured_state,omitempty"`
	Capabilities    map[string]string `json:"capabilities,omitempty"`
	Current         string            `json:"current,omitempty"`
	Warnings        string            `json:"warnings,omitempty"`
	Errors          string            `json:"errors,omitempty"`
	MetadataFields  map[string]string `json:"metadata_fields,omitempty"`
}

// ToolSummary is the per-tool evaluation result included in a StatusReport.
type ToolSummary struct {
	ID              string            `json:"id"`
	DisplayName     string            `json:"display_name"`
	Category        string            `json:"category"`
	Installed       bool              `json:"installed"`
	InstallPath     string            `json:"install_path,omitempty"`
	ConfiguredState ConfiguredState   `json:"configured_state"`
	Configured      bool              `json:"configured"`
	Capabilities    Capability        `json:"capabilities"`
	Current         map[string]string `json:"current,omitempty"`
	Metadata        ToolMetadata      `json:"metadata,omitempty"`
	Warnings        []string          `json:"warnings,omitempty"`
	Errors          []string          `json:"errors,omitempty"`
}

// StatusReport is the top-level structure for status output.
type StatusReport struct {
	SchemaVersion string        `json:"schema_version"`
	GeneratedAt   time.Time     `json:"generated_at"`
	Legend        StatusLegend  `json:"legend,omitempty"`
	Tools         []ToolSummary `json:"tools"`
}

// UseResult is returned by context-switching commands.
type UseResult struct {
	OK      bool              `json:"ok"`
	ToolID  string            `json:"tool_id"`
	Current map[string]string `json:"current,omitempty"`
	Error   string            `json:"error,omitempty"`
}

// NewStatusReport creates a StatusReport pre-populated with the legend.
func NewStatusReport(toolCount int) StatusReport {
	return StatusReport{
		SchemaVersion: SchemaVersionV01,
		Legend: StatusLegend{
			Installed: "Whether the CLI binary was found in PATH.",
			ConfiguredState: map[string]string{
				"yes":     "all-cli found enough local state to treat the tool as configured.",
				"no":      "all-cli checked configuration for this tool and found it missing or empty.",
				"n/a":     "all-cli does not evaluate configuration for this tool.",
				"unknown": "all-cli could not determine configuration state reliably.",
			},
			Capabilities: map[string]string{
				"has_contexts": "The tool exposes a current context-like concept that all-cli can read.",
				"can_switch":   "The tool supports switching context-like state through all-cli commands.",
			},
			Current:  "Tool-specific key/value snapshot of the active context or runtime selection.",
			Warnings: "Non-fatal observations that may affect interpretation, but did not block status collection.",
			Errors:   "Status collection failures or command errors observed while inspecting this tool.",
			MetadataFields: map[string]string{
				"purpose":                    "Short English description of what the tool is for.",
				"configured_when":            "Tool-specific meaning of configured_state=yes.",
				"current_field_descriptions": "English explanation for each known key inside current.",
				"agent_actions":              "Stable high-level actions an agent can reasonably take with this tool.",
				"notes":                      "Short caveats that help interpret ambiguous or non-error states.",
			},
		},
		Tools: make([]ToolSummary, toolCount),
	}
}
