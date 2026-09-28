//go:build windows

package factory

// processAlive stays conservative on Windows: without a POSIX liveness probe
// the lock is always treated as held, so a stale lock needs manual removal.
func processAlive(pid int) bool {
	return true
}
