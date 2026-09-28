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

func TestPickPrimaryHost(t *testing.T) {
	t.Parallel()

	if got := pickPrimaryHost(nil); got != "" {
		t.Fatalf("empty hosts = %q, want empty", got)
	}
	if got := pickPrimaryHost([]Host{{Hostname: "z.example"}, {Hostname: "a.example"}}); got != "a.example" {
		t.Fatalf("expected lexicographic first host, got %q", got)
	}
	if got := pickPrimaryHost([]Host{{Hostname: "z.example"}, {Hostname: "github.com"}, {Hostname: "a.example"}}); got != "github.com" {
		t.Fatalf("expected github.com preferred, got %q", got)
	}
}

func TestUseAccount(t *testing.T) {
	t.Parallel()

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"gh auth switch --hostname github.com --user alice": {},
		},
	})
	if err := a.UseAccount(context.Background(), "github.com", "alice"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUseAccount_RequiresArgs(t *testing.T) {
	t.Parallel()

	a := New(fakeRunner{})
	for _, tc := range [][2]string{{"", "alice"}, {"github.com", ""}, {"  ", "  "}} {
		if err := a.UseAccount(context.Background(), tc[0], tc[1]); err == nil {
			t.Fatalf("expected error for hostname=%q user=%q", tc[0], tc[1])
		}
	}
}

func TestUseAccount_PropagatesSwitchFailure(t *testing.T) {
	t.Parallel()

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"gh auth switch --hostname github.com --user alice": {
				ExitCode: 1,
				Err:      errors.New("exit status 1"),
				Stderr:   "no such user",
			},
		},
	})
	err := a.UseAccount(context.Background(), "github.com", "alice")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "gh auth switch failed (exit=1): no such user") {
		t.Fatalf("unexpected error: %v", err)
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

func TestStatusFromTextGenericErrorIsSanitized(t *testing.T) {
	a := New(fakeRunner{results: map[string]execx.CmdResult{
		"gh auth status": {
			ExitCode: 1,
			Err:      errors.New("exit status 1"),
			Stderr:   "segmentation fault",
		},
	}})

	_, _, errs, err := a.statusFromText(context.Background())
	if err == nil || !strings.Contains(err.Error(), "gh auth status failed") {
		t.Fatalf("err = %v, want gh auth status failure", err)
	}
	if len(errs) != 1 || errs[0] != "segmentation fault" {
		t.Fatalf("errs = %#v", errs)
	}
}

func TestStatusFromTextHostsWithoutSuccess(t *testing.T) {
	text := "github.com\n  X Failed to log in to github.com account bob (token)\n"
	a := New(fakeRunner{results: map[string]execx.CmdResult{
		"gh auth status": {ExitCode: 1, Err: errors.New("exit status 1"), Stderr: text},
	}})

	st, warnings, errs, err := a.statusFromText(context.Background())
	if err != nil || len(warnings) != 0 || len(errs) != 0 {
		t.Fatalf("st=%+v warnings=%v errs=%v err=%v", st, warnings, errs, err)
	}
	if len(st.Hosts) != 1 || st.Hosts[0].Accounts[0].State != "error" {
		t.Fatalf("unexpected status: %#v", st)
	}
}

func TestStatusFromTextFallthroughTreatsEmptyAsUnauthenticated(t *testing.T) {
	a := New(fakeRunner{results: map[string]execx.CmdResult{
		"gh auth status": {Stdout: "no hosts here"},
	}})

	st, warnings, errs, err := a.statusFromText(context.Background())
	if err != nil || len(errs) != 0 {
		t.Fatalf("err=%v errs=%v", err, errs)
	}
	if len(st.Hosts) != 0 || len(warnings) != 1 || !strings.Contains(warnings[0], "unauthenticated") {
		t.Fatalf("st=%+v warnings=%v", st, warnings)
	}
}

func TestSanitizeGHAuthError(t *testing.T) {
	cases := []struct{ in, want string }{
		{"unknown option: --json", unsupportedJSONMessage},
		{"flag provided but not defined: --json", unsupportedJSONMessage},
		{"some crash\nUsage: gh auth login", "some crash"},
		{"  plain error  ", "plain error"},
	}
	for _, tc := range cases {
		if got := sanitizeGHAuthError(tc.in); got != tc.want {
			t.Fatalf("sanitizeGHAuthError(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestFirstToken(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"  ", ""},
		{"bob", "bob"},
		{"bob extra", "bob"},
		{"bob (token)", "bob"},
	}
	for _, tc := range cases {
		if got := firstToken(tc.in); got != tc.want {
			t.Fatalf("firstToken(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestStatusFromTextFallbackOnJSONUnsupported(t *testing.T) {
	a := New(fakeRunner{results: map[string]execx.CmdResult{
		"gh auth status": {
			ExitCode: 1,
			Err:      errors.New("exit status 1"),
			Stderr:   "unknown flag: --json",
		},
	}})

	_, _, errs, err := a.statusFromText(context.Background())
	if err == nil || !strings.Contains(err.Error(), "gh auth status failed") {
		t.Fatalf("err = %v", err)
	}
	if len(errs) != 1 || errs[0] != unsupportedJSONMessage {
		t.Fatalf("errs = %#v, want sanitized unsupported message", errs)
	}
}

func TestStatusFromTextContextErrorPropagates(t *testing.T) {
	a := New(fakeRunner{results: map[string]execx.CmdResult{
		"gh auth status": {ExitCode: -1, Err: context.DeadlineExceeded},
	}})

	_, _, errs, err := a.statusFromText(context.Background())
	if err == nil || !strings.Contains(err.Error(), "gh auth status failed") {
		t.Fatalf("err = %v", err)
	}
	if len(errs) != 1 || errs[0] == "" {
		t.Fatalf("errs = %#v", errs)
	}
}

func TestIsHostHeaderRejectsMarkers(t *testing.T) {
	for _, line := range []string{"✓ github.com", "X github.com", "x github.com", "- github.com", "* github.com"} {
		if isHostHeader(line, strings.TrimSpace(line)) {
			t.Fatalf("isHostHeader(%q) = true, want false", line)
		}
	}
}

func TestExtractAfterMissingNeedle(t *testing.T) {
	if got := extractAfter("no marker here", "Token scopes:"); got != "" {
		t.Fatalf("extractAfter = %q, want empty", got)
	}
}

func TestParseAccountLineEdgeCases(t *testing.T) {
	// State word without a parseable host/login must not produce an account.
	if _, _, ok := parseAccountLine("Failed to log in"); ok {
		t.Fatal("expected no account for bare 'Failed to log in' line")
	}
	// Host-only form ("using token" without account/as) — reachable on the
	// failure-state line since success lines require " account "/" as ".
	acc, host, ok := parseAccountLine("Failed to log in to github.com using token (oauth_token)")
	if !ok || host != "github.com" || acc.Login != "" || acc.State != "error" || acc.TokenSource != "oauth_token" {
		t.Fatalf("host-only parse = %#v host=%q ok=%v", acc, host, ok)
	}
	// " as " separator form.
	acc, host, ok = parseAccountLine("Logged in to ghe.example as bob")
	if !ok || host != "ghe.example" || acc.Login != "bob" {
		t.Fatalf("as-form parse = %#v host=%q ok=%v", acc, host, ok)
	}
}

func TestParseHostAndLoginAndTokenSourceEdges(t *testing.T) {
	if host, login := parseHostAndLogin("no marker line"); host != "" || login != "" {
		t.Fatalf("unexpected host/login %q %q", host, login)
	}
	if src := parseParenTokenSource("no parens"); src != "" {
		t.Fatalf("expected empty token source, got %q", src)
	}
	if src := parseParenTokenSource("reversed )then("); src != "" {
		t.Fatalf("expected empty for reversed parens, got %q", src)
	}
	if src := parseParenTokenSource("token (a/b)"); src != "" {
		t.Fatalf("expected empty for path-like source, got %q", src)
	}
	if src := parseParenTokenSource("token ()"); src != "" {
		t.Fatalf("expected empty for empty parens, got %q", src)
	}
	if got := protocolFromLegacyLine("configured to use ssh protocol"); got != "ssh" {
		t.Fatalf("protocol = %q", got)
	}
	if got := protocolFromLegacyLine("unrelated detail"); got != "" {
		t.Fatalf("expected empty protocol, got %q", got)
	}
}

func TestGHCurrentMultipleHostsAndErrors(t *testing.T) {
	ctx := context.Background()
	twoHosts := `{"hosts":{"github.com":[{"login":"u1","state":"success","active":true}],"ghe.example":[{"login":"u2","state":"success","active":true}]}}`

	a := New(fakeRunner{results: map[string]execx.CmdResult{
		"gh auth status --json hosts": {Stdout: twoHosts},
	}})
	cur, warnings, _, err := a.Current(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if cur["hostname"] != "github.com" || cur["user"] != "u1" {
		t.Fatalf("current = %#v", cur)
	}
	if len(warnings) == 0 {
		t.Fatal("expected multiple-hosts warning")
	}

	// Empty host set returns nil current with no error.
	a = New(fakeRunner{results: map[string]execx.CmdResult{
		"gh auth status --json hosts": {Stdout: `{"hosts":{}}`},
	}})
	cur, _, _, err = a.Current(ctx)
	if err != nil || cur != nil {
		t.Fatalf("empty hosts: cur=%#v err=%v", cur, err)
	}

	// A non-fallback status error propagates through Current and Configured.
	failing := fakeRunner{results: map[string]execx.CmdResult{
		"gh auth status --json hosts": {Err: errors.New("boom"), ExitCode: 1, Stderr: "crash"},
	}}
	a = New(failing)
	if _, _, _, err := a.Current(ctx); err == nil {
		t.Fatal("expected Current to propagate status error")
	}
	if _, _, _, err := a.Configured(ctx); err == nil {
		t.Fatal("expected Configured to propagate status error")
	}
}

func TestConfiguredFalseWhenNoSuccessAccount(t *testing.T) {
	a := New(fakeRunner{results: map[string]execx.CmdResult{
		"gh auth status --json hosts": {Stdout: `{"hosts":{"github.com":[{"login":"u1","state":"error","active":false}]}}`},
	}})
	ok, _, _, err := a.Configured(context.Background())
	if err != nil || ok {
		t.Fatalf("configured=%v err=%v", ok, err)
	}
}
