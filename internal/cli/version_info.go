package cli

import "fmt"

// VersionString returns the full version line for CLI output (e.g. all-cli --version, version subcommand).
func VersionString() string {
	if commit != "" || date != "" {
		return fmt.Sprintf("%s (commit=%s date=%s)", version, commit, date)
	}
	return version
}
