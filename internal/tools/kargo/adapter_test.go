package kargo

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
	return execx.CmdResult{ExitCode: 1, Err: errors.New("unexpected command")}
}

func TestParseConfigView(t *testing.T) {
	stdout := `apiAddress: http://192.168.139.2:8444
bearerToken: '*** REDACTED ***'
insecureSkipTLSVerify: true
kind: CLIConfig
refreshToken: '*** REDACTED ***'
defaultProject: demo
`

	cfg, warnings, errs, err := parseConfigView(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %#v", errs)
	}
	if cfg.APIAddress != "http://192.168.139.2:8444" {
		t.Fatalf("unexpected api address: %q", cfg.APIAddress)
	}
	if cfg.DefaultProject != "demo" {
		t.Fatalf("unexpected default project: %q", cfg.DefaultProject)
	}
}

func TestAdapterSetDefaultProject(t *testing.T) {
	r := fakeRunner{
		results: map[string]execx.CmdResult{
			"kargo config set-project demo": {ExitCode: 0, Err: nil},
		},
	}
	a := New(r)
	if err := a.SetDefaultProject(context.Background(), "demo"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAdapterSetDefaultProject_Unset(t *testing.T) {
	r := fakeRunner{
		results: map[string]execx.CmdResult{
			"kargo config set-project ": {ExitCode: 0, Err: nil},
		},
	}
	a := New(r)
	if err := a.SetDefaultProject(context.Background(), ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAdapterSetDefaultProject_Error(t *testing.T) {
	r := fakeRunner{
		results: map[string]execx.CmdResult{
			"kargo config set-project demo": {ExitCode: 1, Err: errors.New("boom"), Stderr: "bad"},
		},
	}
	a := New(r)
	if err := a.SetDefaultProject(context.Background(), "demo"); err == nil {
		t.Fatalf("expected error")
	}
}
