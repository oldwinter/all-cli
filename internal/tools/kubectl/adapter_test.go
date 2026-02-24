package kubectl

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

func TestAdapterCurrent(t *testing.T) {
	r := fakeRunner{
		results: map[string]execx.CmdResult{
			"kubectl config current-context": {Stdout: "default\n", ExitCode: 0, Err: nil},
			"kubectl config view --minify --output jsonpath={..namespace}{\"\\n\"}": {
				Stdout:   "prod\n",
				ExitCode: 0,
				Err:      nil,
			},
		},
	}
	a := New(r)
	cur, warnings, errs, err := a.Current(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 || len(errs) != 0 {
		t.Fatalf("unexpected warnings/errs: %v %v", warnings, errs)
	}
	if cur["context"] != "default" || cur["namespace"] != "prod" {
		t.Fatalf("unexpected current: %#v", cur)
	}
}

func TestAdapterListContexts(t *testing.T) {
	r := fakeRunner{
		results: map[string]execx.CmdResult{
			"kubectl config get-contexts -o name": {Stdout: "a\nb\n", ExitCode: 0, Err: nil},
		},
	}
	a := New(r)
	contexts, _, _, err := a.ListContexts(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(contexts) != 2 || contexts[0] != "a" || contexts[1] != "b" {
		t.Fatalf("unexpected contexts: %#v", contexts)
	}
}
