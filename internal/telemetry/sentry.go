package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
	secretPattern = regexp.MustCompile(`(?i)\b(token|password|secret|api[_-]?key|authorization)\s*[:=]\s*[^,\s;]+`)
	pathPattern   = regexp.MustCompile(`(?:[A-Za-z]:\\|/)(?:[^/\s:\\]+[/\\]){1,}[^,\s:]*`)
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
				"type":  "CommandError",
				"value": scrubError(commandErr),
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

func scrubError(err error) string {
	if err == nil {
		return ""
	}
	message := secretPattern.ReplaceAllString(err.Error(), "$1=<redacted>")
	message = pathPattern.ReplaceAllString(message, "<path>")
	const maxLength = 8 * 1024
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
