package execx

import (
	"context"
	"strings"
	"testing"
)

func TestRedactSecrets(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain error untouched", in: "error loading config file: yaml: bad line", want: "error loading config file: yaml: bad line"},
		{name: "empty", in: "", want: ""},
		{name: "gh oauth token", in: "oauth_token ghp_abcdefghijklmnopqrstuvwx invalid", want: "oauth_token [redacted] invalid"},
		{name: "github pat", in: "token github_pat_11ABCDEFG0abcdefghijklmnopqrstuvwxyz0123456789 rejected", want: "token [redacted] rejected"},
		{name: "gitlab pat", in: "glpat-abcdefghij1234567890 expired", want: "[redacted] expired"},
		{name: "aws access key", in: "credentials AKIAIOSFODNN7EXAMPLE not found", want: "credentials [redacted] not found"},
		{name: "slack token", in: "xoxb-123456789012-abcdefghijkl auth failed", want: "[redacted] auth failed"},
		{name: "jwt", in: "token eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c", want: "token [redacted]"},
		{name: "bearer header", in: "Authorization: Bearer abcdef1234567890xyz", want: "Authorization: Bearer [redacted]"},
		{name: "kv token", in: "token=supersecretvalue123 rejected", want: "token=[redacted] rejected"},
		{name: "kv api key quoted", in: `api_key="sk_live_abcdef123"`, want: `api_key=[redacted]`},
		{name: "prose token word untouched", in: "token is expired, run login", want: "token is expired, run login"},
		{name: "multiple secrets", in: "AKIAIOSFODNN7EXAMPLE and ghp_abcdefghijklmnopqrstuvwx", want: "[redacted] and [redacted]"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := RedactSecrets(tc.in); got != tc.want {
				t.Fatalf("RedactSecrets(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestDefaultRunnerRedactsStderr(t *testing.T) {
	res := DefaultRunner{}.Run(context.Background(), "sh", "-c",
		`echo '{"data":"ghp_abcdefghijklmnopqrstuvwx"}'; echo "auth failed token ghp_zzzzzzzzzzzzzzzzzzzzzzzz" >&2; exit 1`)
	if strings.Contains(res.Stderr, "ghp_") {
		t.Fatalf("stderr leaked token: %q", res.Stderr)
	}
	if !strings.Contains(res.Stderr, "[redacted]") {
		t.Fatalf("expected [redacted] in stderr, got %q", res.Stderr)
	}
	// stdout stays raw for parsers — adapters rely on exact bytes.
	if !strings.Contains(res.Stdout, "ghp_abcdefghijklmnopqrstuvwx") {
		t.Fatalf("stdout should remain raw for parsing, got %q", res.Stdout)
	}
	if res.ExitCode != 1 {
		t.Fatalf("exit code = %d, want 1", res.ExitCode)
	}
}
