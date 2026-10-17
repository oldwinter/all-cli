package kubectl

import (
	"context"
	"errors"
	"reflect"
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

func TestAdapterConfigured(t *testing.T) {
	tests := []struct {
		name     string
		res      execx.CmdResult
		want     bool
		wantErrs []string
		wantErr  bool
	}{
		{
			name: "contexts listed",
			res:  execx.CmdResult{Stdout: "a\nb\n"},
			want: true,
		},
		{
			name: "no contexts",
			res:  execx.CmdResult{Stdout: "  \n"},
			want: false,
		},
		{
			name:     "get-contexts failure",
			res:      execx.CmdResult{ExitCode: 1, Err: errors.New("exit status 1"), Stderr: "api server unreachable"},
			wantErrs: []string{"api server unreachable"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(fakeRunner{results: map[string]execx.CmdResult{
				"kubectl config get-contexts -o name": tt.res,
			}})
			ok, _, errs, err := a.Configured(context.Background())
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ok != tt.want {
				t.Fatalf("ok = %v, want %v", ok, tt.want)
			}
			if !reflect.DeepEqual(errs, tt.wantErrs) {
				t.Fatalf("errs = %#v, want %#v", errs, tt.wantErrs)
			}
		})
	}
}

func TestAdapterListContextsErrorMessageSelection(t *testing.T) {
	tests := []struct {
		name     string
		res      execx.CmdResult
		wantErrs []string
	}{
		{
			name:     "stderr preferred",
			res:      execx.CmdResult{ExitCode: 1, Err: errors.New("exit status 1"), Stderr: "  config file missing\n", Stdout: "ignored"},
			wantErrs: []string{"config file missing"},
		},
		{
			name:     "stdout fallback",
			res:      execx.CmdResult{ExitCode: 1, Err: errors.New("exit status 1"), Stdout: "error on stdout"},
			wantErrs: []string{"error on stdout"},
		},
		{
			name:     "runner error fallback",
			res:      execx.CmdResult{ExitCode: 1, Err: errors.New(`exec: "kubectl": executable file not found in $PATH`)},
			wantErrs: []string{`exec: "kubectl": executable file not found in $PATH`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(fakeRunner{results: map[string]execx.CmdResult{
				"kubectl config get-contexts -o name": tt.res,
			}})
			contexts, _, errs, err := a.ListContexts(context.Background())
			if err == nil {
				t.Fatal("expected error")
			}
			if contexts != nil {
				t.Fatalf("expected nil contexts, got %#v", contexts)
			}
			if !strings.Contains(err.Error(), "kubectl config get-contexts failed (exit=1)") {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(errs, tt.wantErrs) {
				t.Fatalf("errs = %#v, want %#v", errs, tt.wantErrs)
			}
		})
	}
}

func TestAdapterListContextsSkipsBlankLines(t *testing.T) {
	a := New(fakeRunner{results: map[string]execx.CmdResult{
		"kubectl config get-contexts -o name": {Stdout: "  prod\n\n\tstaging  \n"},
	}})
	contexts, _, _, err := a.ListContexts(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(contexts, []string{"prod", "staging"}) {
		t.Fatalf("unexpected contexts: %#v", contexts)
	}
}

func TestAdapterMutationsSucceed(t *testing.T) {
	tests := []struct {
		name    string
		command string
		invoke  func(Adapter) error
	}{
		{
			name:    "use context",
			command: "kubectl config use-context prod",
			invoke:  func(a Adapter) error { return a.UseContext(context.Background(), "prod") },
		},
		{
			name:    "set namespace for context",
			command: "kubectl config set-context prod --namespace payments",
			invoke:  func(a Adapter) error { return a.SetNamespaceForContext(context.Background(), "prod", "payments") },
		},
		{
			name:    "set namespace for current context",
			command: "kubectl config set-context --current --namespace payments",
			invoke:  func(a Adapter) error { return a.SetNamespaceForCurrentContext(context.Background(), "payments") },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(fakeRunner{results: map[string]execx.CmdResult{
				tt.command: {Stdout: "ok\n"},
			}})
			if err := tt.invoke(a); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestAdapterCurrentCollectsErrors(t *testing.T) {
	tests := []struct {
		name    string
		results map[string]execx.CmdResult
		want    map[string]string
		wantErr int
	}{
		{
			name: "context lookup fails",
			results: map[string]execx.CmdResult{
				"kubectl config current-context":                                     {ExitCode: 1, Err: errors.New("exit status 1"), Stderr: "no current context"},
				`kubectl config view --minify --output jsonpath={..namespace}{"\n"}`: {Stdout: "prod\n"},
			},
			want:    map[string]string{"namespace": "prod"},
			wantErr: 1,
		},
		{
			name: "namespace lookup fails",
			results: map[string]execx.CmdResult{
				"kubectl config current-context": {Stdout: "default\n"},
				`kubectl config view --minify --output jsonpath={..namespace}{"\n"}`: {
					ExitCode: 1,
					Err:      errors.New("exit status 1"),
					Stderr:   "view exploded",
				},
			},
			want:    map[string]string{"context": "default"},
			wantErr: 1,
		},
		{
			name:    "both lookups fail",
			results: map[string]execx.CmdResult{},
			want:    map[string]string{},
			wantErr: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(fakeRunner{results: tt.results})
			cur, _, errs, err := a.Current(context.Background())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(cur, tt.want) {
				t.Fatalf("current = %#v, want %#v", cur, tt.want)
			}
			if len(errs) != tt.wantErr {
				t.Fatalf("errs = %#v, want %d entries", errs, tt.wantErr)
			}
		})
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
