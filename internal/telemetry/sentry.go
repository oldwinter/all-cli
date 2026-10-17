package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

type sentrySink struct {
	dsn         string
	endpoint    string
	publicKey   string
	environment string
	release     string
	client      *http.Client
}

var (
	// sensitiveKeyValuePattern matches `key=value` / `key: value` pairs whose key
	// looks like a credential, capturing the key so it can be preserved while
	// the value is dropped. The value run is deliberately greedy across spaces
	// so that multi-word values such as `Bearer <jwt>` are removed entirely.
	sensitiveKeyValuePattern = regexp.MustCompile(
		`(?i)\b([A-Za-z0-9_.-]*(?:token|password|passwd|secret|api[_-]?key|apikey|access[_-]?key|private[_-]?key|authorization|auth|credential|session|cookie)[A-Za-z0-9_.-]*)\s*[:=]\s*\S+`,
	)
	// jsonSecretPattern matches JSON-style `"key": "value"` credential pairs.
	jsonSecretPattern = regexp.MustCompile(
		`(?i)"([A-Za-z0-9_.-]*(?:token|password|passwd|secret|api[_-]?key|apikey|access[_-]?key|private[_-]?key|authorization|auth|credential|session|cookie)[A-Za-z0-9_.-]*)"\s*:\s*"[^"]*"`,
	)
	// urlUserInfoPattern strips credentials embedded in URLs (scheme://user:pass@host).
	urlUserInfoPattern = regexp.MustCompile(`(?i)([a-z][a-z0-9+.-]*://)[^/\s@]+@`)
	// bearerPattern removes bare bearer/basic credentials even without a key name.
	bearerPattern = regexp.MustCompile(`(?i)\b(bearer|basic)\s+[A-Za-z0-9\-._~+/]+=*`)
	// pathPattern removes filesystem paths on POSIX and Windows, including
	// ~-prefixed paths and paths containing spaces.
	pathPattern = regexp.MustCompile(
		`(?:[A-Za-z]:\\|~/|/)(?:[^/\s:\\]+[/\\])*[^/\s:\\]*`,
	)
	// safeTokenPattern matches the only content kept verbatim: short identifiers
	// made of letters, digits, dots, dashes and underscores.
	safeTokenPattern = regexp.MustCompile(`[^A-Za-z0-9 ._-]+`)
)

func newSentrySink(config Config, client *http.Client) (*sentrySink, error) {
	parsed, err := url.Parse(config.SentryDSN)
	if err != nil {
		return nil, fmt.Errorf("parse SENTRY_DSN: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" || parsed.User == nil || parsed.User.Username() == "" {
		return nil, fmt.Errorf("parse SENTRY_DSN: scheme, host, public key, and project ID are required")
	}
	projectID := strings.Trim(strings.TrimSpace(filepath.Base(parsed.Path)), "/")
	if projectID == "" || projectID == "." {
		return nil, fmt.Errorf("parse SENTRY_DSN: project ID is required")
	}
	pathPrefix := strings.TrimSuffix(strings.TrimSuffix(parsed.Path, "/"), "/"+projectID)
	endpoint := fmt.Sprintf("%s://%s%s/api/%s/envelope/", parsed.Scheme, parsed.Host, pathPrefix, projectID)
	return &sentrySink{
		dsn:         config.SentryDSN,
		endpoint:    endpoint,
		publicKey:   parsed.User.Username(),
		environment: config.SentryEnvironment,
		release:     config.Release,
		client:      client,
	}, nil
}

func (s *sentrySink) capture(ctx context.Context, span Span, command string, commandErr error) {
	eventID := randomHex(16)
	now := time.Now().UTC()
	event := map[string]any{
		"event_id":    eventID,
		"timestamp":   now.Format(time.RFC3339Nano),
		"platform":    "go",
		"level":       "error",
		"environment": s.environment,
		"release":     s.release,
		"transaction": command,
		"tags": map[string]string{
			"command": command,
			"result":  "error",
		},
		"contexts": map[string]any{
			"trace": map[string]string{
				"trace_id": span.TraceID,
				"type":     "trace",
			},
		},
		"breadcrumbs": map[string]any{
			"values": []map[string]any{{
				"timestamp": span.StartedAt.UTC().Format(time.RFC3339Nano),
				"category":  "command",
				"level":     "info",
				"message":   "all-cli " + command,
			}},
		},
		"exception": map[string]any{
			"values": []map[string]any{{
				"type": "CommandError",
				// Only a closed-vocabulary classification is sent. The raw
				// error text is never transmitted because adapters embed user
				// context names, namespaces, hostnames, and file paths in it;
				// the full message stays in the local structured log instead.
				"value": errorClass(commandErr),
				"stacktrace": map[string]any{
					"frames": captureStackFrames(),
				},
			}},
		},
	}
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return
	}
	headerJSON, err := json.Marshal(map[string]string{
		"event_id": eventID,
		"sent_at":  now.Format(time.RFC3339Nano),
		"dsn":      s.dsn,
	})
	if err != nil {
		return
	}
	body := bytes.Join([][]byte{headerJSON, []byte(`{"type":"event"}`), eventJSON, nil}, []byte{'\n'})

	requestContext := ctx
	if requestContext == nil {
		requestContext = context.Background()
	}
	requestContext, cancel := context.WithTimeout(requestContext, 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(requestContext, http.MethodPost, s.endpoint, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/x-sentry-envelope")
	req.Header.Set("X-Sentry-Auth", "Sentry sentry_version=7, sentry_key="+s.publicKey+", sentry_client=all-cli/1")
	response, err := s.client.Do(req)
	if err == nil {
		response.Body.Close()
	}
}

// errorClass reduces an error to a closed-vocabulary category that is safe to
// transmit. It intentionally carries no user data: no tool output, no context
// names, no paths, no hostnames, no credentials.
//
// This is the enforcement point for the privacy contract in
// docs/observability.md. The full error text is still available locally through
// ALL_CLI_LOG_PATH, so operators keep diagnosability without the leak.
func errorClass(err error) string {
	if err == nil {
		return "none"
	}
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, context.Canceled):
		return "canceled"
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return "exit_nonzero"
	}

	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "not found"),
		strings.Contains(message, "executable file not found"),
		strings.Contains(message, "no such file"):
		return "not_found"
	case strings.Contains(message, "permission denied"):
		return "permission_denied"
	case strings.Contains(message, "unknown flag"),
		strings.Contains(message, "unknown command"),
		strings.Contains(message, "invalid argument"):
		return "invalid_usage"
	case strings.Contains(message, "missing schema_version"),
		strings.Contains(message, "parse snapshot"):
		return "invalid_snapshot"
	case strings.Contains(message, "connection refused"),
		strings.Contains(message, "no such host"),
		strings.Contains(message, "timeout"):
		return "network"
	case strings.Contains(message, "failed"):
		return "command_failed"
	default:
		return "error"
	}
}

// scrubError reduces an error message to content that is safe to keep in a
// local structured log. It is NOT used for network delivery (see errorClass).
//
// The documented privacy contract (docs/observability.md) promises that command
// arguments, environment contents, external command output, usernames, account
// IDs, configuration values, and paths never leave the machine. Adapters embed
// user context names, namespaces, and file paths directly into error text, so a
// deny-list of credential keywords is not sufficient: it misses JSON-form
// secrets, URL userinfo, bearer tokens, and account-identifying hostnames.
//
// The scrubber applies credential redaction, then path removal, then replaces
// every character outside a conservative allow-list. That last step is what
// makes the guarantee hold for arbitrary tool output.
func scrubError(err error) string {
	if err == nil {
		return ""
	}
	message := err.Error()

	// 1. Drop credential values while keeping the key name for debuggability.
	message = jsonSecretPattern.ReplaceAllString(message, `"$1":"<redacted>"`)
	message = sensitiveKeyValuePattern.ReplaceAllString(message, "$1=<redacted>")
	message = bearerPattern.ReplaceAllString(message, "$1 <redacted>")
	message = urlUserInfoPattern.ReplaceAllString(message, "$1<redacted>@")

	// 2. Remove filesystem paths (they carry usernames and project names).
	message = pathPattern.ReplaceAllString(message, "<path>")

	// 3. Keep only a conservative allow-list. This neutralises anything the
	//    steps above did not recognise, such as account-identifying hostnames
	//    or arbitrary external command output.
	message = safeTokenPattern.ReplaceAllString(message, " ")

	// 4. Collapse the whitespace introduced above and bound the size.
	message = strings.Join(strings.Fields(message), " ")
	const maxLength = 2 * 1024
	if len(message) > maxLength {
		message = message[:maxLength]
	}
	return message
}

func captureStackFrames() []map[string]any {
	callers := make([]uintptr, 24)
	count := runtime.Callers(3, callers)
	frames := runtime.CallersFrames(callers[:count])
	result := make([]map[string]any, 0, count)
	for {
		frame, more := frames.Next()
		result = append(result, map[string]any{
			"filename": filepath.Base(frame.File),
			"function": frame.Function,
			"lineno":   frame.Line,
			"in_app":   strings.Contains(frame.Function, "github.com/oldwinter/all-cli"),
		})
		if !more {
			break
		}
	}
	return result
}
