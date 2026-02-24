package argocd

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

func TestParseContextTable(t *testing.T) {
	stdout := `CURRENT  NAME                  SERVER
         localhost:8080        localhost:8080
*        localhost:18443       localhost:18443
`

	contexts, warnings, errs, err := parseContextTable(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %#v", errs)
	}
	if len(contexts) != 2 {
		t.Fatalf("expected 2 contexts, got %d", len(contexts))
	}
	if contexts[0].IsCurrent {
		t.Fatalf("expected first context not current: %#v", contexts[0])
	}
	if !contexts[1].IsCurrent || contexts[1].Name != "localhost:18443" {
		t.Fatalf("unexpected current context: %#v", contexts[1])
	}
}

func TestAdapterUseContext(t *testing.T) {
	r := fakeRunner{
		results: map[string]execx.CmdResult{
			"argocd context ctx1": {ExitCode: 0, Err: nil},
		},
	}
	a := New(r)
	if err := a.UseContext(context.Background(), "ctx1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAdapterUseContext_Error(t *testing.T) {
	r := fakeRunner{
		results: map[string]execx.CmdResult{
			"argocd context ctx1": {ExitCode: 1, Err: errors.New("boom"), Stderr: "bad"},
		},
	}
	a := New(r)
	if err := a.UseContext(context.Background(), "ctx1"); err == nil {
		t.Fatalf("expected error")
	}
}
