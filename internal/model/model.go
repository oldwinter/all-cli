package model

import "time"

const SchemaVersionV01 = "v0.1"

type ConfiguredState string

const (
	ConfiguredYes     ConfiguredState = "yes"
	ConfiguredNo      ConfiguredState = "no"
	ConfiguredNA      ConfiguredState = "n/a"
	ConfiguredUnknown ConfiguredState = "unknown"
)

type Capability struct {
	HasContexts bool `json:"has_contexts"`
	CanSwitch   bool `json:"can_switch"`
}

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
	Warnings        []string          `json:"warnings,omitempty"`
	Errors          []string          `json:"errors,omitempty"`
}

type StatusReport struct {
	SchemaVersion string        `json:"schema_version"`
	GeneratedAt   time.Time     `json:"generated_at"`
	Tools         []ToolSummary `json:"tools"`
}

type UseResult struct {
	OK      bool              `json:"ok"`
	ToolID  string            `json:"tool_id"`
	Current map[string]string `json:"current,omitempty"`
	Error   string            `json:"error,omitempty"`
}
