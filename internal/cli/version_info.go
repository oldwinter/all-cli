package cli

import (
	"fmt"
	"runtime/debug"
	"strings"
)

var readBuildInfo = debug.ReadBuildInfo

// VersionString returns the full version line for CLI output (e.g. all-cli --version, version subcommand).
func VersionString() string {
	return formatVersionReport(resolvedVersionReport())
}

func resolvedVersionReport() versionReport {
	report := versionReport{Version: version, Commit: commit, Date: date}
	if report.Version != "dev" || report.Commit != "" || report.Date != "" {
		return report
	}

	info, ok := readBuildInfo()
	if !ok {
		return report
	}
	return resolveVersionReport(report, info)
}

func resolveVersionReport(report versionReport, info *debug.BuildInfo) versionReport {
	if info == nil {
		return report
	}
	if moduleVersion := strings.TrimSpace(info.Main.Version); moduleVersion != "" && moduleVersion != "(devel)" {
		report.Version = moduleVersion
	}
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			report.Commit = strings.TrimSpace(setting.Value)
		case "vcs.time":
			report.Date = strings.TrimSpace(setting.Value)
		}
	}
	return report
}

func formatVersionReport(report versionReport) string {
	if report.Commit != "" || report.Date != "" {
		return fmt.Sprintf("%s (commit=%s date=%s)", report.Version, report.Commit, report.Date)
	}
	return report.Version
}
