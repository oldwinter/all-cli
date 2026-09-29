package execx

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"
)

// CmdResult holds the outcome of an external command execution.
type CmdResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Err      error
}

// OK returns true when the command exited with code 0 and no error.
func (r CmdResult) OK() bool { return r.Err == nil && r.ExitCode == 0 }

// Runner abstracts external command execution for testing.
type Runner interface {
	Run(ctx context.Context, name string, args ...string) CmdResult
}

// gitRedirectEnv names environment variables that repoint which repository a
// spawned `git` operates on. Commands always select their repository with
// -C/Dir; an inherited GIT_DIR or GIT_WORK_TREE would silently bind them to a
// different repo (for example, factory verify could record another repo's
// HEAD), so these never propagate into subprocesses.
var gitRedirectEnv = []string{
	"GIT_DIR",
	"GIT_WORK_TREE",
	"GIT_COMMON_DIR",
	"GIT_INDEX_FILE",
	"GIT_OBJECT_DIRECTORY",
	"GIT_ALTERNATE_OBJECT_DIRECTORIES",
	"GIT_NAMESPACE",
	"GIT_CEILING_DIRECTORIES",
	"GIT_DISCOVERY_ACROSS_FILESYSTEM",
}

// cleanEnv returns the process environment minus variables that would
// redirect repository resolution inside spawned commands.
func cleanEnv() []string {
	env := os.Environ()
	out := env[:0]
	for _, kv := range env {
		key := kv
		if i := strings.IndexByte(kv, '='); i >= 0 {
			key = kv[:i]
		}
		blocked := false
		for _, b := range gitRedirectEnv {
			if key == b {
				blocked = true
				break
			}
		}
		if !blocked {
			out = append(out, kv)
		}
	}
	return out
}

// DefaultRunner executes commands via os/exec.
type DefaultRunner struct{}

func (DefaultRunner) Run(ctx context.Context, name string, args ...string) CmdResult {
	cmd := newCmd(ctx, name, args...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Make timeouts/cancellation detectable even when exec returns "signal: killed".
		if ctxErr := ctx.Err(); ctxErr != nil {
			err = ctxErr
		}
	}
	exitCode := 0
	if err != nil {
		exitCode = 1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
	}

	return CmdResult{
		Stdout:   stdout.String(),
		Stderr:   RedactSecrets(stderr.String()),
		ExitCode: exitCode,
		Err:      err,
	}
}

// LookPath searches for an executable in PATH.
func LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

// TimeoutRunner wraps another Runner with a per-call timeout.
type TimeoutRunner struct {
	Runner  Runner
	Timeout time.Duration
}

func (t TimeoutRunner) Run(ctx context.Context, name string, args ...string) CmdResult {
	if t.Runner == nil {
		return CmdResult{ExitCode: 1, Err: errors.New("nil runner")}
	}
	if t.Timeout <= 0 {
		return t.Runner.Run(ctx, name, args...)
	}
	ctx2, cancel := context.WithTimeout(ctx, t.Timeout)
	defer cancel()
	return t.Runner.Run(ctx2, name, args...)
}
