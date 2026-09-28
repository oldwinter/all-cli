package factory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if os.IsExist(err) {
			meta, _ := os.ReadFile(lockPath)
			return nil, fmt.Errorf("another factory command holds %s (%s); remove it if the holder is gone",
				lockPath, strings.TrimSpace(string(meta)))
		}
		return nil, fmt.Errorf("acquire %s: %w", lockPath, err)
	}
	fmt.Fprintf(f, "pid=%d since=%s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339))
	f.Close()
	return func() { _ = os.Remove(lockPath) }, nil
}
