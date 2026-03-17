package railway

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

func TestCurrentParsesWhoamiJSON(t *testing.T) {
	t.Parallel()

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"railway whoami --json": {
				Stdout: `{"name":"Old Winter","email":"cdd2zju@gmail.com","workspaces":[{"id":"ws_1","name":"Team A"},{"id":"ws_2","name":"Team B"}]}`,
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
	if cur["name"] != "Old Winter" {
		t.Fatalf("expected name, got %#v", cur)
	}
	if cur["email"] != "cdd2zju@gmail.com" {
		t.Fatalf("expected email, got %#v", cur)
	}
	if cur["workspaces_count"] != "2" {
		t.Fatalf("expected workspace count, got %#v", cur)
	}
	if len(warnings) == 0 {
		t.Fatalf("expected warning for multiple workspaces")
	}
}

func TestConfiguredTreatsUnauthorizedAsNotConfigured(t *testing.T) {
	t.Parallel()

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"railway whoami --json": {
				ExitCode: 1,
				Err:      errors.New("exit status 1"),
				Stderr:   "Unauthorized. Please login with `railway login`",
			},
		},
	})

	ok, warnings, errs, err := a.Configured(context.Background())
	if err != nil {
		t.Fatalf("expected unauthorized to be treated as not configured, got error %v", err)
	}
	if ok {
		t.Fatalf("expected ok=false when unauthorized")
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %#v", errs)
	}
}
