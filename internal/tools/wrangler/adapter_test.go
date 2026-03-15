package wrangler

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
	return execx.CmdResult{ExitCode: 1, Err: errors.New("unexpected command")}
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
