package execx

import "strings"

// ErrMessage extracts a human-readable error string from a CmdResult.
// It prefers trimmed Stderr, then falls back to Err.Error().
func ErrMessage(res CmdResult) string {
	if msg := strings.TrimSpace(res.Stderr); msg != "" {
		return msg
	}
	if res.Err != nil {
		return res.Err.Error()
	}
	return ""
}

// StdoutOrStderr returns Stdout when non-blank, otherwise Stderr.
func StdoutOrStderr(res CmdResult) string {
	if strings.TrimSpace(res.Stdout) != "" {
		return res.Stdout
	}
	return res.Stderr
}

// FirstNonEmpty returns the first value whose trimmed form is non-empty.
func FirstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
