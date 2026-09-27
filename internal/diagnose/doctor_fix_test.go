package diagnose

import (
	"github.com/oldwinter/all-cli/internal/execx"
	"strings"
	"testing"
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
