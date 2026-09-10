package gh

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/oldwinter/all-cli/internal/execx"
)

type fakeRunner struct {
	results map[string]execx.CmdResult
}

func (f fakeRunner) Run(_ context.Context, name string, args ...string) execx.CmdResult {
	key := name
	if len(args) > 0 {
		key += " " + strings.Join(args, " ")
	}
	if res, ok := f.results[key]; ok {
		return res
	}
	return execx.CmdResult{ExitCode: 1, Err: errors.New("unexpected command"), Stderr: "unexpected command"}
}

func TestNormalizeHostsAndAccountsOrder(t *testing.T) {
	raw := ghAuthStatus{
		Hosts: map[string][]ghAccount{
			"z.example": {
				{Login: "bob", Active: false, State: "success"},
				{Login: "alice", Active: true, State: "success"},
			},
			"github.com": {
				{Login: "oldwinter", Active: true, State: "success"},
			},
		},
	}

	st := normalize(raw)
	if len(st.Hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(st.Hosts))
	}
	if st.Hosts[0].Hostname != "github.com" {
		t.Fatalf("expected github.com first, got %s", st.Hosts[0].Hostname)
	}
	if st.Hosts[1].Hostname != "z.example" {
		t.Fatalf("expected z.example second, got %s", st.Hosts[1].Hostname)
	}

	accts := st.Hosts[1].Accounts
	if len(accts) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(accts))
	}
	if !accts[0].Active || accts[0].Login != "alice" {
		t.Fatalf("expected active alice first, got %#v", accts[0])
	}
	if accts[1].Active || accts[1].Login != "bob" {
		t.Fatalf("expected bob second, got %#v", accts[1])
	}
}

func TestStatusParsesModernJSON(t *testing.T) {
	t.Parallel()

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"gh auth status --json hosts": {
				Stdout: `{"hosts":{"github.com":[{"login":"oldwinter","active":true,"state":"success","gitProtocol":"ssh","tokenSource":"oauth"}]}}`,
			},
		},
	})

	st, warnings, errs, err := a.Status(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 || len(errs) != 0 {
		t.Fatalf("unexpected diagnostics: %#v %#v", warnings, errs)
	}
	if len(st.Hosts) != 1 || st.Hosts[0].Hostname != "github.com" || st.Hosts[0].Accounts[0].Login != "oldwinter" {
		t.Fatalf("unexpected status: %#v", st)
	}
}

const modernLoggedInText = `github.com
  ✓ Logged in to github.com account oldwinter (keyring)
  - Active account: true
  - Git operations protocol: ssh
  - Token: gho_************************************
  - Token scopes: 'gist', 'read:org', 'repo'
`

const oldLoggedInText = `github.com
  ✓ Logged in to github.com as oldwinter (/home/user/.config/gh/hosts.yml)
  ✓ Git operations for github.com configured to use https protocol.
  ✓ Token: *******************
`

const unknownJSONUsage = `unknown flag: --json

Usage:  gh auth status [flags]

Flags:
  -h, --hostname string   Check a specific hostname's auth status
      --show-token        Display the auth token
`

func TestStatusFallsBackToTextWhenJSONFlagUnknown(t *testing.T) {
	t.Parallel()

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"gh auth status --json hosts": {
				ExitCode: 1,
				Err:      errors.New("exit status 1"),
				Stderr:   unknownJSONUsage,
			},
			"gh auth status": {
				Stderr: modernLoggedInText,
			},
		},
	})

	st, warnings, errs, err := a.Status(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 || len(errs) != 0 {
		t.Fatalf("unexpected diagnostics: %#v %#v", warnings, errs)
	}
	if containsUnknownJSON(warnings, errs) {
		t.Fatalf("usage/unknown flag leaked: %#v %#v", warnings, errs)
	}
	if len(st.Hosts) != 1 {
		t.Fatalf("expected 1 host, got %#v", st)
	}
	acc := st.Hosts[0].Accounts[0]
	if st.Hosts[0].Hostname != "github.com" || acc.Login != "oldwinter" || !acc.Active || acc.State != "success" {
		t.Fatalf("unexpected account: %#v", acc)
	}
	if acc.GitProtocol != "ssh" || acc.TokenSource != "keyring" || !strings.Contains(acc.Scopes, "repo") {
		t.Fatalf("unexpected details: %#v", acc)
	}

	ok, _, _, err := a.Configured(context.Background())
	if err != nil || !ok {
		t.Fatalf("expected configured=yes, ok=%v err=%v", ok, err)
	}
	cur, _, _, err := a.Current(context.Background())
	if err != nil {
		t.Fatalf("current: %v", err)
	}
	if cur["hostname"] != "github.com" || cur["user"] != "oldwinter" {
		t.Fatalf("unexpected current: %#v", cur)
	}
}

func TestStatusFallsBackWhenJSONStdoutIsUsage(t *testing.T) {
	t.Parallel()

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"gh auth status --json hosts": {
				Stdout: unknownJSONUsage,
			},
			"gh auth status": {
				Stdout: oldLoggedInText,
			},
		},
	})

	st, warnings, errs, err := a.Status(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 || len(errs) != 0 {
		t.Fatalf("unexpected diagnostics: %#v %#v", warnings, errs)
	}
	if len(st.Hosts) != 1 || st.Hosts[0].Accounts[0].Login != "oldwinter" {
		t.Fatalf("unexpected status: %#v", st)
	}
	if st.Hosts[0].Accounts[0].GitProtocol != "https" {
		t.Fatalf("expected legacy https protocol, got %#v", st.Hosts[0].Accounts[0])
	}
}

func TestStatusUnauthenticatedOldGH(t *testing.T) {
	t.Parallel()

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"gh auth status --json hosts": {
				ExitCode: 1,
				Err:      errors.New("exit status 1"),
				Stderr:   unknownJSONUsage,
			},
			"gh auth status": {
				ExitCode: 1,
				Err:      errors.New("exit status 1"),
				Stderr:   "You are not logged into any GitHub hosts. To log in, run: gh auth login\n",
			},
		},
	})

	ok, warnings, errs, err := a.Configured(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("expected configured=no")
	}
	if len(errs) != 0 {
		t.Fatalf("did not expect errors, got %#v", errs)
	}
	joined := strings.Join(warnings, "\n")
	if !strings.Contains(joined, "unauthenticated") {
		t.Fatalf("expected unauthenticated warning, got %#v", warnings)
	}
	if containsUnknownJSON(warnings, errs) {
		t.Fatalf("unknown flag leaked: %#v %#v", warnings, errs)
	}
}

func TestStatusOtherJSONErrorsDoNotFallback(t *testing.T) {
	t.Parallel()

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"gh auth status --json hosts": {
				ExitCode: 1,
				Err:      errors.New("exit status 1"),
				Stderr:   "auth broken",
			},
		},
	})

	_, _, errs, err := a.Status(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "gh auth status failed (exit=1)") {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 1 || errs[0] != "auth broken" {
		t.Fatalf("unexpected errs: %#v", errs)
	}
}

func TestStatusPropagatesTimeoutWithoutFallback(t *testing.T) {
	t.Parallel()

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"gh auth status --json hosts": {
				ExitCode: 1,
				Err:      context.DeadlineExceeded,
				Stderr:   "unknown flag: --json",
			},
		},
	})

	_, _, errs, err := a.Status(context.Background())
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(strings.Join(errs, "\n"), "deadline exceeded") && !strings.Contains(err.Error(), "gh auth status failed") {
		t.Fatalf("expected timeout-style failure, got err=%v errs=%#v", err, errs)
	}
}

func TestStatusDoesNotDumpUsageOnFallbackFailure(t *testing.T) {
	t.Parallel()

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"gh auth status --json hosts": {
				ExitCode: 1,
				Err:      errors.New("exit status 1"),
				Stderr:   unknownJSONUsage,
			},
			"gh auth status": {
				ExitCode: 1,
				Err:      errors.New("exit status 1"),
				Stderr:   unknownJSONUsage,
			},
		},
	})

	_, warnings, errs, err := a.Status(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	blob := strings.Join(append(append([]string{}, warnings...), errs...), "\n")
	if strings.Contains(blob, "Usage:") || strings.Contains(blob, "unknown flag: --json") {
		t.Fatalf("usage leaked: %q", blob)
	}
	if !strings.Contains(blob, "2.81") && !strings.Contains(blob, "gh auth status") {
		t.Fatalf("expected upgrade/check-auth message, got %q", blob)
	}
}

func TestParseAuthStatusTextMultipleAccounts(t *testing.T) {
	t.Parallel()

	text := `github.com
  ✓ Logged in to github.com account alice (keyring)
  - Active account: true
  - Git operations protocol: https

  ✓ Logged in to github.com account bob (GH_TOKEN)
  - Active account: false
  - Git operations protocol: ssh

ghe.example.com
  ✓ Logged in to ghe.example.com account bot (keyring)
  - Active account: true
`
	st := parseAuthStatusText(text)
	if len(st.Hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %#v", st)
	}
	if st.Hosts[0].Hostname != "ghe.example.com" {
		t.Fatalf("expected sorted hosts, got %#v", st.Hosts)
	}
	ghHost := st.Hosts[1]
	if len(ghHost.Accounts) != 2 || ghHost.Accounts[0].Login != "alice" || !ghHost.Accounts[0].Active {
		t.Fatalf("expected active alice first: %#v", ghHost.Accounts)
	}
	if ghHost.Accounts[1].Login != "bob" || ghHost.Accounts[1].Active {
		t.Fatalf("expected inactive bob: %#v", ghHost.Accounts[1])
	}
}

func TestSanitizeGHAuthErrorStripsUsage(t *testing.T) {
	t.Parallel()
	got := sanitizeGHAuthError(unknownJSONUsage)
	if strings.Contains(got, "Usage:") || strings.Contains(got, "unknown flag: --json") {
		t.Fatalf("usage not sanitized: %q", got)
	}
	if !strings.Contains(got, "2.81") {
		t.Fatalf("expected upgrade hint, got %q", got)
	}
}

func containsUnknownJSON(groups ...[]string) bool {
	for _, group := range groups {
		for _, item := range group {
			if strings.Contains(item, "unknown flag: --json") || strings.Contains(item, "Usage:") {
				return true
			}
		}
	}
	return false
}
