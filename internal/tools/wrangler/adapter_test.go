package wrangler

import (
	"context"
	"errors"
	"os/exec"
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
	return execx.CmdResult{ExitCode: 1, Err: errors.New("unexpected command")}
}

func TestConfigured(t *testing.T) {
	t.Parallel()

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"wrangler whoami --json": {
				Stdout: `{"loggedIn":true,"accounts":[{"id":"3ba1294bcdfb7a6f8c113ebc120411df"}]}`,
			},
		},
	})

	ok, warnings, errs, err := a.Configured(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected configured=true when logged in")
	}
	if len(warnings) != 0 || len(errs) != 0 {
		t.Fatalf("unexpected diagnostics: %#v %#v", warnings, errs)
	}
}

func TestConfigured_NotLoggedIn(t *testing.T) {
	t.Parallel()

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"wrangler whoami --json": {
				Stdout: `{"loggedIn":false,"accounts":[]}`,
			},
		},
	})

	ok, _, _, err := a.Configured(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected configured=false when logged out")
	}
}

func TestConfigured_PropagatesWhoamiError(t *testing.T) {
	t.Parallel()

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"wrangler whoami --json": {
				ExitCode: 1,
				Err:      exec.ErrNotFound,
			},
		},
	})

	_, _, _, err := a.Configured(context.Background())
	if err == nil {
		t.Fatal("expected fatal whoami error to propagate")
	}
}

func TestWhoamiParsesJSONOutput(t *testing.T) {
	t.Parallel()

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"wrangler whoami --json": {
				Stdout: `{"loggedIn":true,"accounts":[{"id":"3ba1294bcdfb7a6f8c113ebc120411df"},{"id":"2371c3163e63aba96bd280648d9ffffc"},{"id":"0ed12f90b68226a08b1a38f0010e99f2"}]}`,
			},
		},
	})

	got, warnings, errs, err := a.Whoami(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %#v", errs)
	}
	if !got.LoggedIn {
		t.Fatalf("expected logged in, got %#v", got)
	}
	if len(got.AccountIDs) != 3 {
		t.Fatalf("expected 3 account ids, got %#v", got.AccountIDs)
	}
}

func TestWhoamiFallsBackToTextOutput(t *testing.T) {
	t.Parallel()

	text := `
Getting User settings...
👋 You are logged in with an OAuth Token, associated with the email (redacted).
┌──────────────┬──────────────────────────────────┐
│ Account Name │ Account ID                       │
├──────────────┼──────────────────────────────────┤
│ (redacted)   │ 3ba1294bcdfb7a6f8c113ebc120411df │
│ (redacted)   │ 2371c3163e63aba96bd280648d9ffffc │
│ (redacted)   │ 0ed12f90b68226a08b1a38f0010e99f2 │
└──────────────┴──────────────────────────────────┘
`

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"wrangler whoami --json": {
				ExitCode: 1,
				Err:      errors.New("exit status 1"),
				Stderr:   "unknown option: --json",
			},
			"wrangler whoami": {
				Stdout: text,
			},
		},
	})

	got, warnings, errs, err := a.Whoami(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %#v", errs)
	}
	if !got.LoggedIn {
		t.Fatalf("expected logged in, got %#v", got)
	}
	if len(got.AccountIDs) != 3 {
		t.Fatalf("expected 3 ids, got %d: %#v", len(got.AccountIDs), got.AccountIDs)
	}
}

func TestCurrentWarnsOnMultipleAccounts(t *testing.T) {
	t.Parallel()

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"wrangler whoami --json": {
				Stdout: `{"loggedIn":true,"accounts":[{"id":"3ba1294bcdfb7a6f8c113ebc120411df"},{"id":"2371c3163e63aba96bd280648d9ffffc"}]}`,
			},
		},
	})

	cur, warnings, errs, err := a.Current(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %#v", errs)
	}
	if cur["logged_in"] != "yes" || cur["accounts_count"] != "2" {
		t.Fatalf("unexpected current: %#v", cur)
	}
	if _, ok := cur["account_id"]; ok {
		t.Fatalf("did not expect single account_id when multiple accounts exist: %#v", cur)
	}
	if len(warnings) != 1 || warnings[0] != "multiple wrangler accounts detected; no single global default" {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
}

func TestWhoamiPropagatesTimeout(t *testing.T) {
	t.Parallel()

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"wrangler whoami --json": {
				ExitCode: 1,
				Err:      context.DeadlineExceeded,
			},
		},
	})

	_, warnings, errs, err := a.Whoami(context.Background())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
	if len(warnings) != 0 || len(errs) != 0 {
		t.Fatalf("unexpected diagnostics: %#v %#v", warnings, errs)
	}
}

func TestWhoamiMissingBinaryIsError(t *testing.T) {
	t.Parallel()

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"wrangler whoami --json": {
				ExitCode: 1,
				Err:      exec.ErrNotFound,
			},
		},
	})

	got, warnings, errs, err := a.Whoami(context.Background())
	if !errors.Is(err, exec.ErrNotFound) {
		t.Fatalf("expected exec.ErrNotFound, got %v", err)
	}
	if got.LoggedIn {
		t.Fatalf("missing binary must not look logged in: %#v", got)
	}
	if len(warnings) != 0 || len(errs) != 0 {
		t.Fatalf("unexpected diagnostics: %#v %#v", warnings, errs)
	}
}

func TestCurrentMissingBinaryDoesNotReportLoggedOut(t *testing.T) {
	t.Parallel()

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"wrangler whoami --json": {
				ExitCode: 1,
				Err:      exec.ErrNotFound,
			},
		},
	})

	cur, _, _, err := a.Current(context.Background())
	if !errors.Is(err, exec.ErrNotFound) {
		t.Fatalf("expected exec.ErrNotFound, got %v", err)
	}
	if _, ok := cur["logged_in"]; ok {
		t.Fatalf("missing binary must not set logged_in: %#v", cur)
	}
}

func TestCurrentNotLoggedInWhenWhoamiFails(t *testing.T) {
	t.Parallel()

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"wrangler whoami --json": {
				ExitCode: 1,
				Err:      errors.New("exit status 1"),
				Stderr:   "Not logged in",
			},
			"wrangler whoami": {
				ExitCode: 1,
				Err:      errors.New("exit status 1"),
				Stderr:   "You are not logged in",
			},
		},
	})

	cur, warnings, errs, err := a.Current(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cur["logged_in"] != "no" {
		t.Fatalf("expected logged_in=no, got %#v", cur)
	}
	if len(warnings) != 0 || len(errs) != 0 {
		t.Fatalf("unexpected diagnostics: %#v %#v", warnings, errs)
	}
}

func TestParseWhoamiJSONEdges(t *testing.T) {
	t.Parallel()
	if _, ok := parseWhoamiJSON("  "); ok {
		t.Fatal("expected false for blank stdout")
	}
	if _, ok := parseWhoamiJSON("not json"); ok {
		t.Fatal("expected false for invalid JSON")
	}
	w, ok := parseWhoamiJSON(`{"loggedIn":true,"accounts":[{"id":"b"},{"id":"a"},{"id":"a"},{"id":""}]}`)
	if !ok || !w.LoggedIn || len(w.AccountIDs) != 2 || w.AccountIDs[0] != "a" {
		t.Fatalf("whoami = %#v ok=%v", w, ok)
	}
}

func TestWhoamiJSONFallsBackToText(t *testing.T) {
	t.Parallel()
	hex32 := "0123456789abcdef0123456789abcdef"
	a := New(fakeRunner{results: map[string]execx.CmdResult{
		"wrangler whoami --json": {Stdout: "not json"},
		"wrangler whoami":        {Stdout: "You are logged in with account " + hex32},
	}})
	w, _, _, err := a.Whoami(context.Background())
	if err != nil || !w.LoggedIn || len(w.AccountIDs) != 1 || w.AccountIDs[0] != hex32 {
		t.Fatalf("whoami = %#v err=%v", w, err)
	}
}

func TestWhoamiNonFatalTextFallbackNotLoggedIn(t *testing.T) {
	t.Parallel()
	a := New(fakeRunner{results: map[string]execx.CmdResult{
		"wrangler whoami --json": {Err: errors.New("unsupported"), ExitCode: 1},
		"wrangler whoami":        {Err: errors.New("nope"), ExitCode: 1},
	}})
	w, _, _, err := a.Whoami(context.Background())
	if err != nil || w.LoggedIn {
		t.Fatalf("whoami = %#v err=%v", w, err)
	}
}

func TestWranglerCurrentMultiAccountWarning(t *testing.T) {
	t.Parallel()
	a := New(fakeRunner{results: map[string]execx.CmdResult{
		"wrangler whoami --json": {Stdout: `{"loggedIn":true,"accounts":[{"id":"b"},{"id":"a"}]}`},
	}})
	cur, warnings, _, err := a.Current(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if cur["accounts_count"] != "2" || cur["account_id"] != "" {
		t.Fatalf("cur = %#v", cur)
	}
	if len(warnings) == 0 {
		t.Fatal("expected multi-account warning")
	}
}

func TestWranglerCurrentSingleAccountID(t *testing.T) {
	t.Parallel()
	a := New(fakeRunner{results: map[string]execx.CmdResult{
		"wrangler whoami --json": {Stdout: `{"loggedIn":true,"accounts":[{"id":"acct1"}]}`},
	}})
	cur, warnings, _, err := a.Current(context.Background())
	if err != nil || cur["account_id"] != "acct1" || len(warnings) != 0 {
		t.Fatalf("cur=%#v warnings=%#v err=%v", cur, warnings, err)
	}
}
