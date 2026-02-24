package execx

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"time"
)

type CmdResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Err      error
}

func (r CmdResult) OK() bool { return r.Err == nil && r.ExitCode == 0 }

type Runner interface {
	Run(ctx context.Context, name string, args ...string) CmdResult
}

type DefaultRunner struct{}

func (DefaultRunner) Run(ctx context.Context, name string, args ...string) CmdResult {
	cmd := exec.CommandContext(ctx, name, args...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
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
		Stderr:   stderr.String(),
		ExitCode: exitCode,
		Err:      err,
	}
}

func LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

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
