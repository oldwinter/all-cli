package execx

import (
	"context"
	"strings"
	"testing"
)

// Fixtures are assembled at runtime so no literal credential-shaped string
// sits in the source tree (secret scanners would flag them); the runtime
// values exercise the same patterns.
var (
	fakeGHToken   = "ghp_" + strings.Repeat("a", 24)
	fakeGHPAT     = "github_pat_" + "11" + strings.Repeat("x", 40)
	fakeGLPAT     = "glpat-" + strings.Repeat("a", 20)
	fakeAWSKey    = "AKIA" + strings.Repeat("0", 12) + "FAKE"
	fakeSlackTok  = "xoxb-" + strings.Repeat("1", 12) + "-" + strings.Repeat("a", 12)
	fakeJWT       = "eyJ" + strings.Repeat("h", 10) + "." + strings.Repeat("p", 10) + "." + strings.Repeat("s", 10)
	fakeBearer    = strings.Repeat("x", 19)
	fakeKVSecret  = strings.Repeat("s", 19)
	fakeAPIKeyVal = "sk_live_" + strings.Repeat("k", 12)
)

func TestRedactSecrets(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain error untouched", in: "error loading config file: yaml: bad line", want: "error loading config file: yaml: bad line"},
		{name: "empty", in: "", want: ""},
		{name: "gh oauth token", in: "oauth_token " + fakeGHToken + " invalid", want: "oauth_token [redacted] invalid"},
		{name: "github pat", in: "token " + fakeGHPAT + " rejected", want: "token [redacted] rejected"},
		{name: "gitlab pat", in: fakeGLPAT + " expired", want: "[redacted] expired"},
		{name: "aws access key", in: "credentials " + fakeAWSKey + " not found", want: "credentials [redacted] not found"},
		{name: "slack token", in: fakeSlackTok + " auth failed", want: "[redacted] auth failed"},
		{name: "jwt", in: "token " + fakeJWT, want: "token [redacted]"},
		{name: "bearer header", in: "Authorization: Bearer " + fakeBearer, want: "Authorization: Bearer [redacted]"},
		{name: "kv token", in: "token=" + fakeKVSecret + " rejected", want: "token=[redacted] rejected"},
		{name: "kv api key quoted", in: `api_key="` + fakeAPIKeyVal + `"`, want: `api_key=[redacted]`},
		{name: "prose token word untouched", in: "token is expired, run login", want: "token is expired, run login"},
		{name: "multiple secrets", in: fakeAWSKey + " and " + fakeGHToken, want: "[redacted] and [redacted]"},
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
	secret := "ghp_" + strings.Repeat("z", 24)
	res := DefaultRunner{}.Run(context.Background(), "sh", "-c",
		`echo '{"data":"`+secret+`"}'; echo "auth failed token `+secret+`" >&2; exit 1`)
	if strings.Contains(res.Stderr, secret) {
		t.Fatalf("stderr leaked token: %q", res.Stderr)
	}
	if !strings.Contains(res.Stderr, "[redacted]") {
		t.Fatalf("expected [redacted] in stderr, got %q", res.Stderr)
	}
	// stdout stays raw for parsers — adapters rely on exact bytes.
	if !strings.Contains(res.Stdout, secret) {
		t.Fatalf("stdout should remain raw for parsing, got %q", res.Stdout)
	}
	if res.ExitCode != 1 {
		t.Fatalf("exit code = %d, want 1", res.ExitCode)
	}
}
