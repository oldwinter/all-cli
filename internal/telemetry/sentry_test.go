package telemetry

import (
	"errors"
	"testing"
)

func TestScrubErrorRedactsCredentials(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"nil error", nil, ""},
		{
			"authorization header with bearer jwt",
			errors.New(`authorization: Bearer eyJhbGciOiJ9.eyJzdWIiOiIxMjM0In0.c2lnbmF0dXJl`),
			"authorization=<redacted> <redacted>",
		},
		{
			"authorization header case-insensitive",
			errors.New("Authorization=Bearer tok123"),
			"Authorization=<redacted> <redacted>",
		},
		{
			"standalone bearer token",
			errors.New("request failed: Bearer xyz789 was rejected"),
			"request failed: Bearer <redacted> was rejected",
		},
		{
			"basic credentials",
			errors.New("Basic dXNlcjpwYXNz rejected"),
			"Basic <redacted> rejected",
		},
		{
			"token scheme credential",
			errors.New("Token ghp_example123 expired"),
			"Token <redacted> expired",
		},
		{
			"token equals value",
			errors.New("token=abc123 expired"),
			"token=<redacted> expired",
		},
		{
			"api key with colon",
			errors.New("invalid api_key: sk-live-123"),
			"invalid api_key=<redacted>",
		},
		{
			"password",
			errors.New("password: hunter2"),
			"password=<redacted>",
		},
		{
			"bearer token and key-value secret together",
			errors.New("Bearer abc123 and token=zzz"),
			"Bearer <redacted> and token=<redacted>",
		},
		{
			"private path still scrubbed",
			errors.New("open /Users/private/config.yaml: permission denied"),
			"open <path>: permission denied",
		},
		{
			"plain message unchanged",
			errors.New("command failed with exit status 1"),
			"command failed with exit status 1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := scrubError(tt.err); got != tt.want {
				t.Errorf("scrubError(%v) = %q, want %q", tt.err, got, tt.want)
			}
		})
	}
}
