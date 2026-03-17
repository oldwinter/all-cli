package netlify

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

func TestCurrentParsesCurrentUserJSON(t *testing.T) {
	t.Parallel()

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"netlify api getCurrentUser": {
				Stdout: `{"id":"user_123","full_name":"Old Winter","email":"cdd2zju@gmail.com"}`,
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
	if cur["user_id"] != "user_123" {
		t.Fatalf("expected user_id, got %#v", cur)
	}
	if cur["name"] != "Old Winter" {
		t.Fatalf("expected name, got %#v", cur)
	}
	if cur["email"] != "cdd2zju@gmail.com" {
		t.Fatalf("expected email, got %#v", cur)
	}
}

func TestConfiguredTreatsExpiredSessionAsNotConfigured(t *testing.T) {
	t.Parallel()

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"netlify api getCurrentUser": {
				ExitCode: 1,
				Err:      errors.New("exit status 1"),
				Stderr:   "Error: Your session has expired. Please try to re-authenticate by running `netlify logout` and `netlify login`.",
			},
		},
	})

	ok, warnings, errs, err := a.Configured(context.Background())
	if err != nil {
		t.Fatalf("expected expired session to be treated as not configured, got error %v", err)
	}
	if ok {
		t.Fatalf("expected ok=false when session expired")
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %#v", errs)
	}
}
