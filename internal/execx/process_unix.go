//go:build !windows

package execx

import (
	"context"
	"os/exec"
	"syscall"
	"time"
)

// newCmd builds a command that runs in its own process group. Runs are
// shell-wrapped (sh -c), so cancel must kill the whole group — killing only
// the direct child would orphan grandchildren that still hold the output
// pipes, blocking Wait until they exit.
func newCmd(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = cleanEnv()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	cmd.WaitDelay = 5 * time.Second
	return cmd
}
