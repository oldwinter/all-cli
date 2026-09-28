package factory

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// acquireLock takes the repository's factory lock. Mutating commands hold it
// for their whole run so two factory processes cannot interleave backlog
// writes; read-only commands never take it. The lock is a plain file created
// with O_EXCL — a crash can leave it behind, in which case the error message
// names it and it is safe to remove once the holder is gone.
func acquireLock(root string) (func(), error) {
	dir := filepath.Join(root, ".factory")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create %s: %w", dir, err)
	}
	lockPath := filepath.Join(dir, "lock")
	release, err := createLock(lockPath)
	if err == nil {
		return release, nil
	}
	if !os.IsExist(err) {
		return nil, fmt.Errorf("acquire %s: %w", lockPath, err)
	}
	meta, _ := os.ReadFile(lockPath)
	if lockHolderDead(string(meta)) {
		// The recorded holder is gone — the lock is stale from a crash
		// (e.g. SIGKILL). Reclaim it and retry once.
		_ = os.Remove(lockPath)
		if release, err := createLock(lockPath); err == nil {
			return release, nil
		}
	}
	return nil, fmt.Errorf("another factory command holds %s (%s); remove it if the holder is gone",
		lockPath, strings.TrimSpace(string(meta)))
}

func createLock(lockPath string) (func(), error) {
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(f, "pid=%d since=%s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339))
	f.Close()
	return func() { _ = os.Remove(lockPath) }, nil
}

// lockHolderDead reports whether the lock's recorded pid no longer exists.
// Signal 0 probes liveness: ESRCH means the process is gone; EPERM or nil
// means something still owns it (pid reuse fails safe — we keep refusing).
func lockHolderDead(meta string) bool {
	for _, field := range strings.Fields(meta) {
		if pid, ok := strings.CutPrefix(field, "pid="); ok {
			if n, err := strconv.Atoi(pid); err == nil && n > 0 {
				return syscall.Kill(n, 0) == syscall.ESRCH
			}
		}
	}
	return false
}
