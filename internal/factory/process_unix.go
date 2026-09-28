//go:build !windows

package factory

import "syscall"

// processAlive reports whether pid exists. Signal 0 probes liveness: ESRCH
// means the process is gone; EPERM or nil means something still owns it —
// pid reuse fails safe (a live stranger keeps the lock refusal).
func processAlive(pid int) bool {
	return syscall.Kill(pid, 0) != syscall.ESRCH
}
