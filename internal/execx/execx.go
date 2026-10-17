package execx

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"regexp"
	"time"
)

// waitDelay bounds how long Run waits for the child's stdout/stderr pipes to
// close after the process exits or the context is cancelled.
//
// When Stdout/Stderr are not *os.File, os/exec creates a pipe and a copying
// goroutine, and cmd.Run() waits for that goroutine. If the child spawned a
// grandchild that inherited the write end (shim scripts, .cmd wrappers, kubectl
// plugins, docker credential helpers), the pipe stays open after the direct
// child is killed and Run() blocks until the grandchild exits. Without this
// setting a `--timeout 200ms` invocation was observed to hang for minutes.
// cmd.WaitDelay is the standard-library knob for exactly this case.
const waitDelay = 2 * time.Second

// maxOutputBytes caps how much of a child's stdout/stderr is retained in memory.
// A pathological or misbehaving tool must not be able to exhaust the CLI's memory.
const maxOutputBytes = 1 << 20 // 1 MiB, matching the snapshot stdin limit

// ansiEscapePattern matches ANSI/VT escape sequences so captured output can be
// stored as clean text. External CLIs colour their stderr even when the reader
// is not a terminal, which would otherwise leak escape bytes into --json output
// and into human-readable error sections despite NO_COLOR.
var ansiEscapePattern = regexp.MustCompile(
	// CSI sequences: colours, cursor movement, erase-in-line, cursor hide/show.
	`\x1b\[[0-9;?]*[ -/]*[@-~]` +
		// OSC sequences terminated by BEL or ST.
		`|\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)` +
		// Remaining two-byte escapes.
		`|\x1b[@-Z\\-_]`,
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

// DefaultRunner executes commands via os/exec.
type DefaultRunner struct{}

func (DefaultRunner) Run(ctx context.Context, name string, args ...string) CmdResult {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.WaitDelay = waitDelay

	var stdout cappedBuffer
	var stderr cappedBuffer
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
		Stdout:   stripANSI(stdout.String()),
		Stderr:   stripANSI(stderr.String()),
		ExitCode: exitCode,
		Err:      err,
	}
}

// stripANSI removes ANSI escape sequences from captured command output.
func stripANSI(s string) string {
	if s == "" {
		return s
	}
	return ansiEscapePattern.ReplaceAllString(s, "")
}

// cappedBuffer is an io.Writer that keeps at most max bytes and remembers that
// it truncated, so callers can tell a complete capture from a partial one.
type cappedBuffer struct {
	buf       bytes.Buffer
	truncated bool
}

func (c *cappedBuffer) Write(p []byte) (int, error) {
	originalLen := len(p)
	if remaining := maxOutputBytes - c.buf.Len(); remaining > 0 {
		if len(p) > remaining {
			c.buf.Write(p[:remaining])
			c.truncated = true
		} else {
			c.buf.Write(p)
		}
	} else if originalLen > 0 {
		c.truncated = true
	}
	// Always report a full write: the child must not fail because we stopped
	// retaining bytes, and callers only read the retained prefix.
	return originalLen, nil
}

func (c *cappedBuffer) String() string {
	if !c.truncated {
		return c.buf.String()
	}
	return c.buf.String() + "\n[all-cli: output truncated at 1 MiB]"
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
