//go:build !windows

package factory

import (
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
)

// acquireLock holds an exclusive non-blocking flock on .factory/lock. The
// kernel releases it when the process exits — even on SIGKILL — so a crash
// can never leave a stale lock, and acquisition is atomic (no
// remove-then-create race). The file itself stays behind as a breadcrumb.
func acquireLock(root string) (func(), error) {
	lockPath, err := lockDir(root)
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("acquire %s: %w", lockPath, err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		meta, _ := io.ReadAll(f)
		f.Close()
		return nil, fmt.Errorf("another factory command holds %s (%s)",
			lockPath, strings.TrimSpace(string(meta)))
	}
	writeLockMeta(f)
	return func() { _ = f.Close() }, nil
}
