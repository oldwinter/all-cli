package k9s

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

func TestParseK9sInfoConfig(t *testing.T) {
	tests := []struct {
		name   string
		stdout string
		want   string
	}{
		{
			name:   "normal output",
			stdout: " ____  __.________\n|    |/ _/   __   \\\nConfig:  /home/user/.config/k9s/config.yaml\n",
			want:   "/home/user/.config/k9s/config.yaml",
		},
		{
			name:   "with ANSI escape codes",
			stdout: "\x1b[36mConfig:\x1b[0m /tmp/k9s.yaml\n",
			want:   "/tmp/k9s.yaml",
		},
		{
			name:   "no config line",
			stdout: "some random output\nother line\n",
			want:   "",
		},
		{
			name:   "empty input",
			stdout: "",
			want:   "",
		},
		{
			name:   "config with spaces",
			stdout: "Config:    /path/with spaces/config.yaml\n",
			want:   "/path/with spaces/config.yaml",
		},
	}
	for _, tt := range tests {
		got := parseK9sInfoConfig(tt.stdout)
		if got != tt.want {
			t.Errorf("%s: parseK9sInfoConfig = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestAdapterCurrent_MergesKubectlAndK9sInfo(t *testing.T) {
	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"kubectl config current-context": {
				Stdout: "prod-cluster\n",
			},
			"kubectl config view --minify --output jsonpath={..namespace}{\"\\n\"}": {
				Stdout: "payments\n",
			},
			"k9s info": {
				Stdout: "Config:  /tmp/k9s.yaml\n",
			},
		},
	})

	cur, warnings, errs, err := a.Current(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 || len(errs) != 0 {
		t.Fatalf("unexpected diagnostics: %#v %#v", warnings, errs)
	}
	if cur["context"] != "prod-cluster" || cur["namespace"] != "payments" || cur["config"] != "/tmp/k9s.yaml" {
		t.Fatalf("unexpected current map: %#v", cur)
	}
}

func TestAdapterCurrent_WarnsWhenNamespaceMissing(t *testing.T) {
	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"kubectl config current-context": {
				Stdout: "prod-cluster\n",
			},
			"kubectl config view --minify --output jsonpath={..namespace}{\"\\n\"}": {
				Stdout: "",
			},
			"k9s info": {
				Stdout: "Config:  /tmp/k9s.yaml\n",
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
	if cur["context"] != "prod-cluster" || cur["config"] != "/tmp/k9s.yaml" {
		t.Fatalf("unexpected current map: %#v", cur)
	}
	if _, ok := cur["namespace"]; ok {
		t.Fatalf("did not expect namespace entry, got %#v", cur)
	}
	if len(warnings) != 1 || warnings[0] != "k9s context detected via kubeconfig; namespace not set in kubeconfig" {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
}
