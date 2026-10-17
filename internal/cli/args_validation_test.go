package cli

import (
	"bytes"
	"io"
	"testing"
)

// TestZeroArgCommandsRejectExtraArgs locks in the argument-validation rule.
//
// Before this was enforced, cobra's default ArbitraryArgs silently accepted a
// typo'd positional argument: `all-cli status extra` printed a full report and
// exited 0. For a tool whose README markets it to scripts and agents, a silent
// success on a mistyped command is worse than an error.
func TestZeroArgCommandsRejectExtraArgs(t *testing.T) {
	t.Parallel()

	paths := [][]string{
		{"status"},
		{"diagnose"},
		{"doctor"},
		{"snapshot"},
		{"version"},
		{"options"},
		{"surprise"},
		{"schema"},
		{"docker", "status"},
		{"docker", "current"},
		{"docker", "list"},
		{"docker", "update"},
		{"kubectl", "status"},
		{"kubectl", "current"},
		{"kubectl", "list"},
		{"gh", "status"},
		{"gh", "list"},
		{"glab", "status"},
		{"glab", "list"},
		{"argocd", "status"},
		{"argocd", "list"},
		{"kargo", "status"},
		{"kargo", "current"},
		{"aws", "status"},
		{"aws", "list"},
		{"aliyun", "status"},
		{"aliyun", "list"},
		{"mise", "status"},
		{"mise", "current"},
		{"k9s", "status"},
		{"k9s", "current"},
		{"wrangler", "status"},
		{"wrangler", "current"},
	}

	for _, path := range paths {
		path := path
		t.Run(joinPath(path), func(t *testing.T) {
			t.Parallel()

			args := append(append([]string{}, path...), "unexpected-positional-arg")
			err := executeForArgsTest(args)
			if err == nil {
				t.Fatalf("%v: expected an error for an unexpected positional argument, got nil", path)
			}
		})
	}
}

// TestCommandsWithOptionalArgsStillAcceptThem guards the other side of the
// change: commands that legitimately take a positional argument must keep
// working.
func TestCommandsWithOptionalArgsStillAcceptThem(t *testing.T) {
	t.Parallel()

	tests := [][]string{
		{"catalog", "kubernetes"},
		{"describe", "kubectl"},
		{"kubectl", "use", "some-context"},
		{"docker", "use", "some-context"},
	}

	for _, args := range tests {
		args := args
		t.Run(joinPath(args), func(t *testing.T) {
			t.Parallel()

			// The command may still fail (no real tool behind it), but it must
			// not fail with an argument-validation error.
			err := executeForArgsTest(args)
			if err != nil && isArgsValidationError(err) {
				t.Fatalf("%v: unexpected argument-validation error: %v", args, err)
			}
		})
	}
}

// executeForArgsTest runs the root command with output discarded.
func executeForArgsTest(args []string) error {
	root := NewRootCommand()
	root.SetArgs(args)
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SilenceErrors = true
	root.SilenceUsage = true
	return root.Execute()
}

// isArgsValidationError reports whether err came from cobra's Args validator.
func isArgsValidationError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	for _, needle := range []string{
		"unknown command",
		"accepts at most",
		"accepts between",
		"accepts ",
		"requires at least",
		"required flag",
	} {
		if bytes.Contains([]byte(msg), []byte(needle)) {
			return true
		}
	}
	return false
}

func joinPath(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += "_"
		}
		out += p
	}
	return out
}
