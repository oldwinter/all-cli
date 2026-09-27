package kargo

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/oldwinter/all-cli/internal/execx"
)

const configViewCmd = "kargo config view"

func viewRunner(res execx.CmdResult) fakeRunner {
	return fakeRunner{results: map[string]execx.CmdResult{configViewCmd: res}}
}

func TestParseConfigViewCases(t *testing.T) {
	tests := []struct {
		name   string
		stdout string
		want   Config
	}{
		{name: "empty output", stdout: "", want: Config{}},
		{
			name:   "blank and colon-less lines are skipped",
			stdout: "\n   \nnot a key value line\napiAddress: https://kargo.example\n",
			want:   Config{APIAddress: "https://kargo.example"},
		},
		{
			name:   "double quotes and surrounding whitespace are trimmed",
			stdout: "  apiAddress :  \"https://kargo.example:443\"  \n\tdefaultProject:\t'demo'\n",
			want:   Config{APIAddress: "https://kargo.example:443", DefaultProject: "demo"},
		},
		{
			name:   "tokens and unknown keys are ignored",
			stdout: "bearerToken: secret\nrefreshToken: secret\nkind: CLIConfig\ninsecureSkipTLSVerify: true\n",
			want:   Config{},
		},
		{
			name:   "empty values stay empty",
			stdout: "apiAddress:\ndefaultProject: ''\n",
			want:   Config{},
		},
		{
			name:   "later keys override earlier ones",
			stdout: "defaultProject: first\ndefaultProject: second\n",
			want:   Config{DefaultProject: "second"},
		},
		{
			name:   "keys are case sensitive",
			stdout: "APIAddress: https://ignored\nDefaultProject: ignored\n",
			want:   Config{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, warnings, errs, err := parseConfigView(tt.stdout)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(warnings) != 0 || len(errs) != 0 {
				t.Fatalf("unexpected warnings=%#v errs=%#v", warnings, errs)
			}
			if cfg != tt.want {
				t.Fatalf("config = %#v, want %#v", cfg, tt.want)
			}
		})
	}
}

func TestParseConfigViewScannerError(t *testing.T) {
	stdout := "apiAddress: https://kargo.example\nbearerToken: " + strings.Repeat("x", 70*1024) + "\ndefaultProject: demo\n"

	cfg, _, errs, err := parseConfigView(stdout)
	if err == nil {
		t.Fatalf("expected error for overlong line")
	}
	if len(errs) != 1 || errs[0] != err.Error() {
		t.Fatalf("errs = %#v, want [%q]", errs, err.Error())
	}
	if cfg.APIAddress != "https://kargo.example" {
		t.Fatalf("expected values before the overlong line to be kept, got %#v", cfg)
	}
	if cfg.DefaultProject != "" {
		t.Fatalf("expected parsing to stop at the overlong line, got %#v", cfg)
	}
}

func TestAdapterViewConfig(t *testing.T) {
	tests := []struct {
		name     string
		res      execx.CmdResult
		want     Config
		wantErrs []string
		wantErr  string
	}{
		{
			name: "success",
			res:  execx.CmdResult{Stdout: "apiAddress: https://kargo.example\ndefaultProject: demo\n"},
			want: Config{APIAddress: "https://kargo.example", DefaultProject: "demo"},
		},
		{
			name:     "failure reports trimmed stderr",
			res:      execx.CmdResult{ExitCode: 2, Err: errors.New("exit status 2"), Stderr: "  not logged in\n"},
			wantErrs: []string{"not logged in"},
			wantErr:  "kargo config view failed (exit=2)",
		},
		{
			name:     "failure falls back to error text when stderr is blank",
			res:      execx.CmdResult{ExitCode: 1, Err: errors.New("executable not found"), Stderr: " \n"},
			wantErrs: []string{"executable not found"},
			wantErr:  "kargo config view failed (exit=1)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, warnings, errs, err := New(viewRunner(tt.res)).ViewConfig(context.Background())
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("err = %v, want %q", err, tt.wantErr)
			}
			if len(warnings) != 0 {
				t.Fatalf("unexpected warnings: %#v", warnings)
			}
			if len(errs) != len(tt.wantErrs) || (len(errs) > 0 && !reflect.DeepEqual(errs, tt.wantErrs)) {
				t.Fatalf("errs = %#v, want %#v", errs, tt.wantErrs)
			}
			if cfg != tt.want {
				t.Fatalf("config = %#v, want %#v", cfg, tt.want)
			}
		})
	}
}

func TestAdapterConfigured(t *testing.T) {
	failure := execx.CmdResult{ExitCode: 1, Err: errors.New("boom"), Stderr: "no config"}
	tests := []struct {
		name     string
		res      execx.CmdResult
		want     bool
		wantErrs []string
		wantErr  bool
	}{
		{name: "api address set", res: execx.CmdResult{Stdout: "apiAddress: https://kargo.example\n"}, want: true},
		{name: "only project set", res: execx.CmdResult{Stdout: "defaultProject: demo\n"}, want: false},
		{name: "empty config", res: execx.CmdResult{}, want: false},
		{name: "command fails", res: failure, wantErrs: []string{"no config"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, errs, err := New(viewRunner(tt.res)).Configured(context.Background())
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("configured = %v, want %v", got, tt.want)
			}
			if len(errs) != len(tt.wantErrs) || (len(errs) > 0 && !reflect.DeepEqual(errs, tt.wantErrs)) {
				t.Fatalf("errs = %#v, want %#v", errs, tt.wantErrs)
			}
		})
	}
}

func TestAdapterCurrent(t *testing.T) {
	tests := []struct {
		name    string
		res     execx.CmdResult
		want    map[string]string
		wantErr bool
	}{
		{
			name: "address and project",
			res:  execx.CmdResult{Stdout: "apiAddress: https://kargo.example\ndefaultProject: demo\n"},
			want: map[string]string{"api_address": "https://kargo.example", "project": "demo"},
		},
		{
			name: "address only",
			res:  execx.CmdResult{Stdout: "apiAddress: https://kargo.example\n"},
			want: map[string]string{"api_address": "https://kargo.example"},
		},
		{
			name: "project only",
			res:  execx.CmdResult{Stdout: "defaultProject: demo\n"},
			want: map[string]string{"project": "demo"},
		},
		{
			name: "nothing configured",
			res:  execx.CmdResult{Stdout: "kind: CLIConfig\n"},
			want: map[string]string{},
		},
		{
			name:    "command fails",
			res:     execx.CmdResult{ExitCode: 1, Err: errors.New("boom")},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, errs, err := New(viewRunner(tt.res)).Current(context.Background())
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				if got != nil {
					t.Fatalf("expected nil map on failure, got %#v", got)
				}
				if !reflect.DeepEqual(errs, []string{"boom"}) {
					t.Fatalf("errs = %#v, want [\"boom\"]", errs)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("current = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestAdapterSetDefaultProjectErrorMessage(t *testing.T) {
	tests := []struct {
		name string
		res  execx.CmdResult
		want string
	}{
		{
			name: "stderr is preferred",
			res:  execx.CmdResult{ExitCode: 3, Err: errors.New("exit status 3"), Stderr: " project not found \n"},
			want: "kargo config set-project failed (exit=3): project not found",
		},
		{
			name: "error text when stderr is blank",
			res:  execx.CmdResult{ExitCode: 1, Err: errors.New("executable not found")},
			want: "kargo config set-project failed (exit=1): executable not found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := fakeRunner{results: map[string]execx.CmdResult{"kargo config set-project demo": tt.res}}
			err := New(r).SetDefaultProject(context.Background(), "demo")
			if err == nil || err.Error() != tt.want {
				t.Fatalf("err = %v, want %q", err, tt.want)
			}
		})
	}
}
