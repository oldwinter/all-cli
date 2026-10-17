// Package schemas exposes the JSON Schemas bundled with all-cli.
package schemas

import (
	"embed"
	"fmt"
	"strings"
)

// Stable schema names accepted by Read and by `all-cli schema <name>`.
const (
	Status       = "status"
	Diagnostic   = "diagnostic"
	DoctorFix    = "doctor-fix"
	FixPlan      = "fix-plan"
	SnapshotDiff = "snapshot-diff"
)

var schemaFiles = map[string]string{
	Status:       "status-report-v0.1.json",
	Diagnostic:   "diagnostic-report-v0.1.json",
	DoctorFix:    "doctor-fix-report-v0.1.json",
	FixPlan:      "fix-plan-v0.1.json",
	SnapshotDiff: "snapshot-diff-v0.1.json",
}

//go:embed *.json
var files embed.FS

// Names returns the stable names accepted by Read, in a deterministic order.
func Names() []string {
	return []string{Status, Diagnostic, DoctorFix, FixPlan, SnapshotDiff}
}

// Read returns one bundled JSON Schema by its stable name.
func Read(name string) ([]byte, error) {
	fileName, ok := schemaFiles[name]
	if !ok {
		return nil, fmt.Errorf("unknown schema %q; expected one of: %s", name, strings.Join(Names(), ", "))
	}
	return files.ReadFile(fileName)
}
