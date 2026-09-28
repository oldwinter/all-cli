package cli

import (
	"runtime/debug"
	"testing"
)

func TestResolveVersionReportNilInfo(t *testing.T) {
	report := versionReport{Version: "v1"}
	if got := resolveVersionReport(report, nil); got != report {
		t.Fatalf("resolveVersionReport(nil info) = %+v", got)
	}
	if got := resolveVersionReport(report, &debug.BuildInfo{}); got != report {
		t.Fatalf("resolveVersionReport(empty info) = %+v", got)
	}
}
