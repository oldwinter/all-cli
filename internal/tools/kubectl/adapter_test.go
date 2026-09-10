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

func TestAdapterErrorsIncludeRunnerCause(t *testing.T) {
	tests := []struct {
		name    string
		command string
		invoke  func(Adapter) error
		want    string
	}{
		{
			name:    "use context",
			command: "kubectl config use-context prod",
			invoke: func(a Adapter) error {
				return a.UseContext(context.Background(), "prod")
			},
			want: "kubectl config use-context",
		},
		{
			name:    "set namespace for context",
			command: "kubectl config set-context prod --namespace payments",
			invoke: func(a Adapter) error {
				return a.SetNamespaceForContext(context.Background(), "prod", "payments")
			},
			want: "kubectl config set-context",
		},
		{
			name:    "set namespace for current context",
			command: "kubectl config set-context --current --namespace payments",
			invoke: func(a Adapter) error {
				return a.SetNamespaceForCurrentContext(context.Background(), "payments")
			},
			want: "kubectl config set-context --current",
		},
		{
			name:    "get current context",
			command: "kubectl config current-context",
			invoke: func(a Adapter) error {
				_, err := a.currentContext(context.Background())
				return err
			},
			want: "kubectl config current-context",
		},
		{
			name:    "get current namespace",
			command: "kubectl config view --minify --output jsonpath={..namespace}{\"\\n\"}",
			invoke: func(a Adapter) error {
				_, err := a.currentNamespace(context.Background())
				return err
			},
			want: "kubectl config view (namespace)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cause := errors.New(`exec: "kubectl": executable file not found in $PATH`)
			a := New(fakeRunner{results: map[string]execx.CmdResult{
				tt.command: {ExitCode: 1, Err: cause},
			}})

			err := tt.invoke(a)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) || !strings.Contains(err.Error(), cause.Error()) {
				t.Fatalf("error = %q, want command and runner cause", err)
			}
		})
	}
}
