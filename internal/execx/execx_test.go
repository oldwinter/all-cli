package execx

import (
	"context"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

// echoCommand returns a portable command that echoes its arguments to stdout.
// Windows has no `echo` executable (it is a shell builtin), so use cmd /c.
func echoCommand(args ...string) (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", append([]string{"/c", "echo"}, args...)
	}
	return "echo", args
}

// failingCommand returns a portable command that exits non-zero.
func failingCommand() (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/c", "exit 1"}
	}
	return "false", nil
}

// sleepingCommand returns a portable command that sleeps long enough to be
// interrupted by a cancelled context.
func sleepingCommand() (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/c", "ping -n 30 127.0.0.1 >nul"}
	}
	return "sleep", []string{"10"}
}

func TestDefaultRunnerRun(t *testing.T) {
	r := DefaultRunner{}
	name, args := echoCommand("hello")
	res := r.Run(context.Background(), name, args...)
	if res.Err != nil {
		t.Fatalf("unexpected error: %v", res.Err)
	}
	if !res.OK() {
		t.Fatalf("expected OK, got exit code %d", res.ExitCode)
	}
	// Trim because Windows `echo` and POSIX `echo` may differ in trailing space.
	if got := strings.TrimSpace(res.Stdout); got != "hello" {
		t.Fatalf("stdout = %q, want %q", got, "hello")
	}
}

func TestDefaultRunnerRunFailure(t *testing.T) {
	r := DefaultRunner{}
	name, args := failingCommand()
	res := r.Run(context.Background(), name, args...)
	if res.Err == nil {
		t.Fatal("expected error for failing command")
	}
	if res.ExitCode == 0 {
		t.Fatalf("exit code = %d, want non-zero", res.ExitCode)
	}
	if res.OK() {
		t.Fatal("expected OK() to be false")
	}
}

func TestDefaultRunnerContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := DefaultRunner{}
	name, args := sleepingCommand()
	res := r.Run(ctx, name, args...)
	if res.Err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestTimeoutRunnerNilRunner(t *testing.T) {
	tr := TimeoutRunner{Runner: nil, Timeout: time.Second}
	name, args := echoCommand("test")
	res := tr.Run(context.Background(), name, args...)
	if res.Err == nil {
		t.Fatal("expected error for nil runner")
	}
}

func TestTimeoutRunnerZeroTimeout(t *testing.T) {
	inner := DefaultRunner{}
	tr := TimeoutRunner{Runner: inner, Timeout: 0}
	name, args := echoCommand("hello")
	res := tr.Run(context.Background(), name, args...)
	if !res.OK() {
		t.Fatalf("expected OK with zero timeout, got error: %v", res.Err)
	}
}

func TestTimeoutRunnerWithTimeout(t *testing.T) {
	inner := DefaultRunner{}
	tr := TimeoutRunner{Runner: inner, Timeout: 5 * time.Second}
	name, args := echoCommand("hello")
	res := tr.Run(context.Background(), name, args...)
	if !res.OK() {
		t.Fatalf("expected OK, got error: %v", res.Err)
	}
}

func TestLookPathExisting(t *testing.T) {
	// Use an executable guaranteed to exist wherever the tests run.
	name := "go"
	if runtime.GOOS == "windows" {
		name = "cmd"
	}
	path, err := LookPath(name)
	if err != nil {
		t.Fatalf("expected to find %q: %v", name, err)
	}
	if path == "" {
		t.Fatal("expected non-empty path")
	}
}

func TestLookPathNonExistent(t *testing.T) {
	_, err := LookPath("definitely-not-a-real-command-12345")
	if err == nil {
		t.Fatal("expected error for non-existent command")
	}
}

func TestCmdResultOK(t *testing.T) {
	if !(CmdResult{ExitCode: 0}).OK() {
		t.Fatal("expected OK for zero exit code and nil error")
	}
	if (CmdResult{ExitCode: 1}).OK() {
		t.Fatal("expected not OK for non-zero exit code")
	}
}

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "no escapes here", "no escapes here"},
		{"empty", "", ""},
		{"colour", "\x1b[31mError: failed\x1b[0m", "Error: failed"},
		{"cursor hide", "\x1b[?25lprogress\x1b[?25h", "progress"},
		{"erase line", "\r\x1b[Kdone", "\rdone"},
		{"osc title", "\x1b]0;title\x07text", "text"},
		{"nested", "\x1b[1m\x1b[32mok\x1b[0m", "ok"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stripANSI(tt.in); got != tt.want {
				t.Fatalf("stripANSI(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestCappedBufferTruncates(t *testing.T) {
	var buf cappedBuffer
	payload := strings.Repeat("x", maxOutputBytes+512)
	n, err := buf.Write([]byte(payload))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// A short write would make the child see an I/O error, so Write must
	// always report the full length.
	if n != len(payload) {
		t.Fatalf("Write returned %d, want %d", n, len(payload))
	}
	got := buf.String()
	if !strings.Contains(got, "output truncated") {
		t.Fatal("expected truncation marker")
	}
	if strings.Count(got, "x") != maxOutputBytes {
		t.Fatalf("retained %d bytes, want %d", strings.Count(got, "x"), maxOutputBytes)
	}
}

func TestCappedBufferKeepsSmallOutputIntact(t *testing.T) {
	var buf cappedBuffer
	if _, err := buf.Write([]byte("small")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := buf.String(); got != "small" {
		t.Fatalf("String() = %q, want %q", got, "small")
	}
}

func TestDefaultRunnerStripsANSIFromCapturedOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("relies on a POSIX printf")
	}
	r := DefaultRunner{}
	res := r.Run(context.Background(), "printf", `\033[31mred\033[0m`)
	if res.Err != nil {
		t.Fatalf("unexpected error: %v", res.Err)
	}
	if strings.ContainsRune(res.Stdout, 0x1b) {
		t.Fatalf("expected escape bytes to be stripped, got %q", res.Stdout)
	}
	if res.Stdout != "red" {
		t.Fatalf("stdout = %q, want %q", res.Stdout, "red")
	}
}

// TestDefaultRunnerWaitDelayBoundsGrandchildPipe is a regression test for the
// unbounded-hang bug: when a child spawns a grandchild that inherits stdout,
// os/exec's pipe-copy goroutine keeps Run blocked after the direct child exits.
// Without cmd.WaitDelay this was measured at 6s+ (and minutes in the worst
// case); with it, Run returns shortly after the context deadline.
func TestDefaultRunnerWaitDelayBoundsGrandchildPipe(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("uses a Windows .cmd shim to detach a grandchild holding the pipe")
	}
	dir := t.TempDir()
	shim := dir + `\hold.cmd`
	// `start /b` detaches a grandchild that inherits the inherited stdout pipe
	// and sleeps ~10s, while the shim itself exits immediately.
	script := "@echo off\r\nstart \"\" /b cmd /c \"ping -n 10 127.0.0.1 >nul\"\r\nexit /b 0\r\n"
	if err := os.WriteFile(shim, []byte(script), 0o600); err != nil {
		t.Fatalf("write shim: %v", err)
	}

	r := DefaultRunner{}
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	start := time.Now()
	res := r.Run(ctx, shim)
	elapsed := time.Since(start)

	// Generous bound: slow CI may take seconds, but an unbounded regression
	// would blow well past this.
	if elapsed > 6*time.Second {
		t.Fatalf("Run took %v; cmd.WaitDelay should have bounded the pipe teardown", elapsed)
	}
	if res.Err == nil {
		t.Fatal("expected a timeout error, got nil")
	}
}
