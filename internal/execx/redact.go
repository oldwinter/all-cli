package execx

import (
	"regexp"
	"strings"
)

// secretPatterns match credential-shaped strings that CLIs sometimes echo into
// stderr (auth failures, verbose errors). all-cli surfaces stderr into status
// errors, diagnostics, and factory run logs, so these are replaced with
// [redacted] at capture time. Stdout is left raw: adapters parse it as data.
var secretPatterns = []*regexp.Regexp{
	// Well-known token prefixes (GitHub, GitLab, AWS, Slack).
	regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{16,}`),
	regexp.MustCompile(`github_pat_[A-Za-z0-9_]{16,}`),
	regexp.MustCompile(`glpat-[A-Za-z0-9_-]{15,}`),
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
	regexp.MustCompile(`xox[baprs]-[A-Za-z0-9-]{10,}`),
	// JSON Web Tokens: header.payload.signature, each base64url.
	regexp.MustCompile(`eyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}`),
	// Authorization headers and key=value style secrets in error text.
	regexp.MustCompile(`(?i)(bearer\s+)[A-Za-z0-9._~+/-]{8,}`),
}

// kvSecretPattern covers key=value/key:value secrets where the value may be
// bare, single-quoted, double-quoted (with escapes), or embedded in JSON-ish
// output like {"api_key":"..."} where a quote sits between key and separator.
// The unquoted alternative stops at , } ] " so neighbouring fields are not
// consumed; \S+ is the last-resort for values starting with an unclosed quote.
var kvSecretPattern = regexp.MustCompile(`(?i)\b((?:oauth_?token|api[_-]?key|secret|password|access[_-]?token|token)["']?\s*[=:]\s*)("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'|[^\s",}\]]+|\S+)`)

const redactedPlaceholder = "[redacted]"

// RedactSecrets replaces credential-shaped strings with [redacted]. It is
// applied to captured stderr before it can flow into reports or run logs.
func RedactSecrets(s string) string {
	for _, re := range secretPatterns {
		if re.NumSubexp() > 0 {
			s = re.ReplaceAllString(s, "${1}"+redactedPlaceholder)
		} else {
			s = re.ReplaceAllString(s, redactedPlaceholder)
		}
	}
	// Keep the value's quote characters when present so key="..." and
	// "key":"..." stay structurally valid after redaction.
	s = kvSecretPattern.ReplaceAllStringFunc(s, func(m string) string {
		sub := kvSecretPattern.FindStringSubmatch(m)
		prefix, val := sub[1], sub[2]
		if q := val[:1]; (q == `"` || q == "'") && strings.HasSuffix(val, q) {
			return prefix + q + redactedPlaceholder + q
		}
		return prefix + redactedPlaceholder
	})
	return s
}
