package factory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
)

// CheckResult records one executed verification check.
type CheckResult struct {
	Index    int
	Total    int
	Command  string
	ExitCode int
	Err      error
	Log      string
}

// OK reports whether the check passed.
func (r CheckResult) OK() bool { return r.Err == nil && r.ExitCode == 0 }

// Runner executes item checks through internal/execx with a per-check timeout
// and writes combined output into .factory/run/<id>/.
type Runner struct {
	Exec     execx.Runner
	Root     string
	RunDir   string
	Timeout  time.Duration
	Now      func() time.Time
	Head     func() string
	OnResult func(CheckResult)
}

// NewRunner returns a Runner with production defaults for the repo at root.
func NewRunner(root string) Runner {
	return Runner{
		Exec:    execx.TimeoutRunner{Runner: execx.DefaultRunner{}, Timeout: 10 * time.Minute},
		Root:    root,
		RunDir:  filepath.Join(root, ".factory", "run"),
		Timeout: 10 * time.Minute,
		Now:     time.Now,
		Head:    gitHead(root, execx.TimeoutRunner{Runner: execx.DefaultRunner{}, Timeout: 15 * time.Second}),
	}
}

func gitHead(root string, runner execx.Runner) func() string {
	return func() string {
		res := runner.Run(context.Background(), "git", "-C", root, "rev-parse", "--short=7", "HEAD")
		if !res.OK() {
			return ""
		}
		return strings.TrimSpace(res.Stdout)
	}
}

func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

// RunChecks executes each check from the item's root directory. It stops at
// the first failing check and returns the results produced so far.
func (r Runner) RunChecks(ctx context.Context, item *WorkItem) ([]CheckResult, error) {
	stamp := r.Now().UTC().Format("20060102T150405Z")
	logDir := filepath.Join(r.RunDir, item.ID)
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, fmt.Errorf("create run dir %s: %w", logDir, err)
	}
	var results []CheckResult
	for i, check := range item.Checks {
		logName := fmt.Sprintf("%s-check-%d.log", stamp, i)
		logPath := filepath.Join(logDir, logName)
		wrapped := "cd " + shellQuote(r.Root) + " && " + check
		res := r.Exec.Run(ctx, "sh", "-c", wrapped)
		result := CheckResult{Command: check, ExitCode: res.ExitCode, Err: res.Err}
		var logBody strings.Builder
		fmt.Fprintf(&logBody, "$ %s\n(exit=%d)\n\n--- stdout ---\n%s\n--- stderr ---\n%s\n",
			check, res.ExitCode, res.Stdout, res.Stderr)
		if res.Err != nil {
			fmt.Fprintf(&logBody, "--- error ---\n%v\n", res.Err)
		}
		if err := os.WriteFile(logPath, []byte(logBody.String()), 0o644); err != nil {
			return results, fmt.Errorf("write check log: %w", err)
		}
		if rel, err := filepath.Rel(r.Root, logPath); err == nil {
			result.Log = filepath.ToSlash(rel)
		} else {
			result.Log = logPath
		}
		result.Index = i
		result.Total = len(item.Checks)
		if r.OnResult != nil {
			r.OnResult(result)
		}
		results = append(results, result)
		if !result.OK() {
			break
		}
	}
	return results, nil
}
