package vercel

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

func TestCurrentReadsWhoamiAndCurrentScope(t *testing.T) {
	t.Parallel()

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"vercel whoami --format json": {
				Stdout: `{"username":"oldwinter","email":"cdd2zju@gmail.com","name":null}`,
			},
			"vercel teams ls --format json --next 0": {
				Stdout: "Fetching teams\nFetching user information\n" +
					`{"teams":[{"id":"team_123","slug":"oldwinters-projects","name":"oldwinter's projects","current":true},{"id":"team_456","slug":"other-team","name":"Other Team","current":false}]}`,
			},
		},
	})

	cur, warnings, errs, err := a.Current(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %#v", errs)
	}
	if cur["user"] != "oldwinter" {
		t.Fatalf("expected user=oldwinter, got %#v", cur)
	}
	if cur["email"] != "cdd2zju@gmail.com" {
		t.Fatalf("expected email, got %#v", cur)
	}
	if cur["scope"] != "oldwinters-projects" {
		t.Fatalf("expected scope from current team, got %#v", cur)
	}
}

func TestConfiguredTreatsMissingLoginAsNotConfigured(t *testing.T) {
	t.Parallel()

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"vercel whoami --format json": {
				ExitCode: 1,
				Err:      errors.New("exit status 1"),
				Stderr:   "Error: No existing credentials found. Please run `vercel login`",
			},
		},
	})

	ok, warnings, errs, err := a.Configured(context.Background())
	if err != nil {
		t.Fatalf("expected missing login to be treated as not configured, got error %v", err)
	}
	if ok {
		t.Fatalf("expected ok=false when login is missing")
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %#v", errs)
	}
}
