package diagnose

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
)

func TestDoctorCommandFailureFallsBackToStdoutAndExitCode(t *testing.T) {
	tests := []struct {
		name string
		res  execx.CmdResult
		want string
	}{
		{name: "stdout", res: execx.CmdResult{ExitCode: 1, Stdout: "  Error:\n  formula not found  "}, want: "Error: formula not found"},
		{name: "exit code", res: execx.CmdResult{ExitCode: 7}, want: "command exited with code 7"},
		{name: "truncated", res: execx.CmdResult{ExitCode: 1, Stderr: strings.Repeat("x", 300)}, want: strings.Repeat("x", 237) + "..."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := doctorCommandFailure(tt.res); got != tt.want {
				t.Fatalf("doctorCommandFailure() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDoctorFixFailureRedactsStdoutSecret(t *testing.T) {
	secret := "token=" + strings.Repeat("s", 20)
	runner := &recordingRunner{results: map[string]execx.CmdResult{
		"brew install aliyun-cli": {ExitCode: 1, Stdout: "install failed " + secret, Err: errors.New("exit 1")},
	}}
	report := model.DiagnosticReport{Diagnostics: []model.DiagnosticItem{{
		ID: "aliyun.not_installed", RelatedTool: "aliyun",
		SuggestedActions: []model.SuggestedAction{{ID: "install_tool"}},
	}}}
	out := RunDoctorFixes(context.Background(), runner, report, DoctorFixOptions{Installer: "brew"}, func(string) (string, error) { return "/fake/brew", nil })
	if len(out.Items) != 1 || out.Items[0].Status != model.DoctorFixFailed {
		t.Fatalf("unexpected items: %#v", out.Items)
	}
	if strings.Contains(out.Items[0].Reason, secret) {
		t.Fatalf("stdout secret leaked into reason: %q", out.Items[0].Reason)
	}
}
