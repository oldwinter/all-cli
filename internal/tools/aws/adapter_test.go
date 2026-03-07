package aws

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
	key := name + " " + strings.Join(args, " ")
	if r, ok := f.results[key]; ok {
		return r
	}
	return execx.CmdResult{ExitCode: 1, Err: errors.New("unexpected command"), Stderr: "unexpected command"}
}

func TestAdapterCurrent_PreservesLookupDiagnostics(t *testing.T) {
	t.Setenv("AWS_PROFILE", "prod")
	t.Setenv("AWS_REGION", "")
	t.Setenv("AWS_DEFAULT_REGION", "")
	t.Setenv("AWS_DEFAULT_OUTPUT", "")

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"aws configure get region --profile prod": {
				ExitCode: 1,
				Err:      errors.New("exit status 1"),
				Stderr:   "missing region",
			},
			"aws configure get output --profile prod": {
				ExitCode: 0,
				Stdout:   "json\n",
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
	if cur["profile"] != "prod" {
		t.Fatalf("expected profile to be preserved, got %#v", cur)
	}
	if cur["output"] != "json" {
		t.Fatalf("expected successful output lookup to be preserved, got %#v", cur)
	}
	if _, ok := cur["region"]; ok {
		t.Fatalf("expected missing region after failed lookup, got %#v", cur)
	}
	if len(errs) == 0 {
		t.Fatalf("expected lookup errors to be returned")
	}
	if errs[0] != "missing region" {
		t.Fatalf("unexpected errs: %#v", errs)
	}
}

func TestAdapterCurrent_PrefersEnvironment(t *testing.T) {
	t.Setenv("AWS_PROFILE", "dev")
	t.Setenv("AWS_REGION", "us-west-2")
	t.Setenv("AWS_DEFAULT_OUTPUT", "yaml")

	a := New(fakeRunner{})

	cur, warnings, errs, err := a.Current(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 || len(errs) != 0 {
		t.Fatalf("unexpected warnings/errs: %#v %#v", warnings, errs)
	}
	if cur["profile"] != "dev" || cur["region"] != "us-west-2" || cur["output"] != "yaml" {
		t.Fatalf("unexpected current: %#v", cur)
	}
}
