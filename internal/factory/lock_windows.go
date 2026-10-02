//go:build windows

package factory

import (
	"fmt"
	"os"
	"strings"
)

// acquireLock falls back to O_EXCL creation on Windows, which has no POSIX
// flock. A crashed command leaves the file behind; remove it once the holder
// is gone (the error message names the recorded pid and start time).
func acquireLock(root string) (func(), error) {
	lockPath, err := lockDir(root)
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if os.IsExist(err) {
			meta, _ := os.ReadFile(lockPath)
			return nil, fmt.Errorf("another factory command holds %s (%s); remove it if the holder is gone",
				lockPath, strings.TrimSpace(string(meta)))
		}
		return nil, fmt.Errorf("acquire %s: %w", lockPath, err)
	}
	writeLockMeta(f)
	f.Close()
	return func() { _ = os.Remove(lockPath) }, nil
}
