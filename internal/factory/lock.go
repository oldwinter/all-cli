package factory

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// acquireLock takes the repository's factory lock. Mutating commands hold it
// for their whole run so two factory processes cannot interleave backlog
// writes; read-only commands never take it. Platform files implement the
// actual mechanism (flock on Unix, O_EXCL on Windows).

func lockDir(root string) (string, error) {
	dir := filepath.Join(root, ".factory")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create %s: %w", dir, err)
	}
	return filepath.Join(dir, "lock"), nil
}

// writeLockMeta records the holder for diagnostics — the file content is
// informational only; exclusion comes from the platform lock, not the bytes.
func writeLockMeta(f *os.File) {
	_ = f.Truncate(0)
	fmt.Fprintf(f, "pid=%d since=%s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339))
}
