//go:build windows

package execx

import (
	"context"
	"os/exec"
	"time"
)

// newCmd keeps the default CommandContext kill semantics on Windows: the OS
// has no POSIX process groups, so cancellation terminates the direct child.
func newCmd(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.WaitDelay = 5 * time.Second
	return cmd
}
