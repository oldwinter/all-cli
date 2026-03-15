package cli

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
)

const glabStatusAllOutput = `gitlab.example.com
  ✓ Logged in to gitlab.example.com as oldwinter (/Users/me/.config/glab-cli/config.yml)
  ✓ Git operations for gitlab.example.com configured to use ssh protocol.
  ✓ API calls for gitlab.example.com are made over https protocol.
  ✓ REST API Endpoint: https://gitlab.example.com/api/v4/
  ✓ GraphQL Endpoint: https://gitlab.example.com/api/graphql/
  ✓ Token found: **************************
`

func TestGLabStatusJSON(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"glab auth status": {
				Stdout: glabStatusAllOutput,
			},
			"glab config get host": {
				Stdout: "gitlab.example.com\n",
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newGLabCommand(opts, runner), "status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	var got struct {
		Current map[string]string `json:"current"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if got.Current["effective_host"] != "gitlab.example.com" || got.Current["global_host"] != "gitlab.example.com" || got.Current["user"] != "oldwinter" {
		t.Fatalf("unexpected current: %#v", got.Current)
	}
}

func TestGLabStatusPlain(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"glab auth status": {
				Stdout: glabStatusAllOutput,
			},
			"glab config get host": {
				Stdout: "gitlab.example.com\n",
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newGLabCommand(opts, runner), "status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, needle := range []string{"effective_host: gitlab.example.com", "global_host: gitlab.example.com", "user: oldwinter"} {
		if !strings.Contains(stdout, needle) {
			t.Fatalf("expected %q in stdout, got:\n%s", needle, stdout)
		}
	}
}

func TestGLabListJSON(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"glab auth status --all": {
				Stdout: glabStatusAllOutput,
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newGLabCommand(opts, runner), "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	var got struct {
		Instances []map[string]any `json:"instances"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if len(got.Instances) != 1 || got.Instances[0]["host"] != "gitlab.example.com" {
		t.Fatalf("unexpected payload: %#v", got)
	}
}

func TestGLabListPlainWithoutUserAndWithError(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"glab auth status --all": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stdout: `gitlab.example.com
  x gitlab.example.com: API call failed: EOF
  ! No token found (checked config file, keyring, and environment variables).
`,
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newGLabCommand(opts, runner), "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "gitlab.example.com ok=no") {
		t.Fatalf("unexpected stdout:\n%s", stdout)
	}
	if !strings.Contains(stderr, "warning: glab auth status --all exited with code 1") {
		t.Fatalf("expected warning in stderr, got %q", stderr)
	}
}

func TestGLabListJSONWithWarnings(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"glab auth status --all": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stdout: `gitlab.example.com
  x gitlab.example.com: API call failed: EOF
`,
			},
		},
	}

	stdout, _, err := executeTestCommand(t, newGLabCommand(opts, runner), "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got struct {
		Warnings []string `json:"warnings"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if len(got.Warnings) == 0 {
		t.Fatalf("expected warnings, got %#v", got)
	}
}

func TestGLabListPlainError(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"glab auth status --all": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "ERROR",
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newGLabCommand(opts, runner), "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	for _, needle := range []string{
		"warning: glab auth status --all exited with code 1",
		"error: glab auth status --all returned no instances",
	} {
		if !strings.Contains(stderr, needle) {
			t.Fatalf("expected %q in stderr, got %q", needle, stderr)
		}
	}
}

func TestGLabListJSONError(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"glab auth status --all": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "ERROR",
			},
		},
	}

	stdout, _, err := executeTestCommand(t, newGLabCommand(opts, runner), "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got struct {
		Warnings []string `json:"warnings"`
		Errors   []string `json:"errors"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if len(got.Warnings) != 2 ||
		got.Warnings[0] != "glab output parsed but no instances detected" ||
		got.Warnings[1] != "glab auth status --all exited with code 1" ||
		len(got.Errors) != 1 || got.Errors[0] != "glab auth status --all returned no instances" {
		t.Fatalf("unexpected payload: %#v", got)
	}
}

func TestGLabStatusPlainWithDiagnostics(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"glab auth status": {
				ExitCode: 1,
				Stdout:   glabStatusAllOutput,
				Err:      assertError("exit status 1"),
			},
			"glab config get host": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "config missing",
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newGLabCommand(opts, runner), "status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "effective_host: gitlab.example.com") {
		t.Fatalf("unexpected stdout:\n%s", stdout)
	}
	if !strings.Contains(stderr, "warning: glab auth status exited with code 1") || !strings.Contains(stderr, "error: config missing") {
		t.Fatalf("expected diagnostics in stderr, got %q", stderr)
	}
}

func TestGLabStatusPlainWithoutUserOrGlobalHost(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"glab auth status": {
				Stdout: `gitlab.example.com
  ✓ Logged in to gitlab.example.com (/Users/me/.config/glab-cli/config.yml)
`,
			},
			"glab config get host": {
				Stdout: "\n",
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newGLabCommand(opts, runner), "status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if strings.Contains(stdout, "user:") || strings.Contains(stdout, "global_host:") || !strings.Contains(stdout, "effective_host: gitlab.example.com") {
		t.Fatalf("unexpected stdout:\n%s", stdout)
	}
}

func TestGLabUsePlainSuccess(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"glab config set host gitlab.example.com": {},
			"glab config get host":                    {Stdout: "gitlab.example.com\n"},
			"glab auth status":                        {Stdout: glabStatusAllOutput},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newGLabCommand(opts, runner), "use", "gitlab.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "set glab global host to gitlab.example.com") {
		t.Fatalf("unexpected stdout:\n%s", stdout)
	}
}

func TestGLabListPlain(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"glab auth status --all": {
				Stdout: glabStatusAllOutput,
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newGLabCommand(opts, runner), "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "gitlab.example.com ok=yes user=oldwinter") {
		t.Fatalf("unexpected stdout:\n%s", stdout)
	}
}

func TestGLabUseJSONSuccess(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"glab config set host gitlab.example.com": {},
			"glab config get host": {
				Stdout: "gitlab.example.com\n",
			},
			"glab auth status": {
				Stdout: glabStatusAllOutput,
			},
		},
	}

	stdout, _, err := executeTestCommand(t, newGLabCommand(opts, runner), "use", "gitlab.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got model.UseResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if !got.OK || got.ToolID != "glab" || got.Current["global_host"] != "gitlab.example.com" {
		t.Fatalf("unexpected use result: %#v", got)
	}
}

func TestGLabUseJSONFailure(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"glab config set host gitlab.example.com": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "write failed",
			},
		},
	}

	stdout, _, err := executeTestCommand(t, newGLabCommand(opts, runner), "use", "gitlab.example.com")
	if err == nil {
		t.Fatal("expected error")
	}
	var got model.UseResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if got.OK || !strings.Contains(got.Error, "write failed") {
		t.Fatalf("unexpected use result: %#v", got)
	}
}
