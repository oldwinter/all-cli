package cli

import (
	"os"
	"strings"
)

// isTerminal reports whether f is an interactive character device (TTY).
func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// ansiDisabledByEnv reports NO_COLOR / TERM=dumb (https://no-color.org/).
func ansiDisabledByEnv() bool {
	if os.Getenv("NO_COLOR") != "" {
		return true
	}
	if strings.EqualFold(os.Getenv("TERM"), "dumb") {
		return true
	}
	return false
}

// terminalAnsiEnabled is true when it is reasonable to emit ANSI styling on f.
func terminalAnsiEnabled(f *os.File) bool {
	if !isTerminal(f) {
		return false
	}
	return !ansiDisabledByEnv()
}

func ciEnvSet() bool {
	return strings.TrimSpace(os.Getenv("CI")) != ""
}

func allCliNoProgressEnvSet() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("ALL_CLI_NO_PROGRESS"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// statusSpinnerEnabled controls the status command progress line on stderr.
// Disabled when not a TTY, when ANSI is opted out, in CI, or when ALL_CLI_NO_PROGRESS is set.
func statusSpinnerEnabled() bool {
	if !terminalAnsiEnabled(os.Stderr) {
		return false
	}
	if ciEnvSet() {
		return false
	}
	if allCliNoProgressEnvSet() {
		return false
	}
	return true
}
