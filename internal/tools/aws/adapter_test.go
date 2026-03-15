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

func TestAdapterCurrent_IgnoresUnsetOptionalOutput(t *testing.T) {
	t.Setenv("AWS_PROFILE", "default")
	t.Setenv("AWS_REGION", "")
	t.Setenv("AWS_DEFAULT_REGION", "")
	t.Setenv("AWS_DEFAULT_OUTPUT", "")

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"aws configure get region --profile default": {
				ExitCode: 0,
				Stdout:   "us-east-1\n",
			},
			"aws configure get output --profile default": {
				ExitCode: 1,
				Err:      errors.New("exit status 1"),
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
		t.Fatalf("expected no errors for unset optional output, got %#v", errs)
	}
	if cur["profile"] != "default" || cur["region"] != "us-east-1" {
		t.Fatalf("unexpected current: %#v", cur)
	}
	if _, ok := cur["output"]; ok {
		t.Fatalf("expected output to be omitted when unset, got %#v", cur)
	}
}

func TestAdapterListProfiles_Success(t *testing.T) {
	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"aws configure list-profiles": {
				Stdout: "default\nprod\n",
			},
		},
	})

	profiles, warnings, errs, err := a.ListProfiles(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 || len(errs) != 0 {
		t.Fatalf("unexpected diagnostics: %#v %#v", warnings, errs)
	}
	if len(profiles) != 2 || profiles[0] != "default" || profiles[1] != "prod" {
		t.Fatalf("unexpected profiles: %#v", profiles)
	}
}

func TestAdapterListProfiles_Failure(t *testing.T) {
	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"aws configure list-profiles": {
				ExitCode: 1,
				Err:      errors.New("exit status 1"),
				Stderr:   "broken config",
			},
		},
	})

	profiles, warnings, errs, err := a.ListProfiles(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if len(profiles) != 0 {
		t.Fatalf("expected no profiles, got %#v", profiles)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	if len(errs) != 1 || errs[0] != "broken config" {
		t.Fatalf("unexpected errs: %#v", errs)
	}
}
