package cli

import (
	"os"
	"strings"
	"testing"
)

func TestAnsiDisabledByEnv(t *testing.T) {
	t.Parallel()

	oldNO, oldTERM := os.Getenv("NO_COLOR"), os.Getenv("TERM")
	t.Cleanup(func() {
		restoreEnv(t, "NO_COLOR", oldNO)
		restoreEnv(t, "TERM", oldTERM)
	})

	if err := os.Unsetenv("NO_COLOR"); err != nil {
		t.Fatal(err)
	}
	if err := os.Unsetenv("TERM"); err != nil {
		t.Fatal(err)
	}
	if ansiDisabledByEnv() {
		t.Fatal("expected ansi not disabled with empty env")
	}

	if err := os.Setenv("NO_COLOR", "1"); err != nil {
		t.Fatal(err)
	}
	if !ansiDisabledByEnv() {
		t.Fatal("expected NO_COLOR to disable ansi")
	}
	if err := os.Unsetenv("NO_COLOR"); err != nil {
		t.Fatal(err)
	}

	if err := os.Setenv("TERM", "dumb"); err != nil {
		t.Fatal(err)
	}
	if !ansiDisabledByEnv() {
		t.Fatal("expected TERM=dumb to disable ansi")
	}
}

func TestStatusSpinnerEnvHelpers(t *testing.T) {
	t.Parallel()

	oldCI := os.Getenv("CI")
	oldProg := os.Getenv("ALL_CLI_NO_PROGRESS")
	t.Cleanup(func() {
		restoreEnv(t, "CI", oldCI)
		restoreEnv(t, "ALL_CLI_NO_PROGRESS", oldProg)
	})

	if err := os.Unsetenv("CI"); err != nil {
		t.Fatal(err)
	}
	if ciEnvSet() {
		t.Fatal("expected CI unset to be false")
	}
	if err := os.Setenv("CI", "true"); err != nil {
		t.Fatal(err)
	}
	if !ciEnvSet() {
		t.Fatal("expected CI set")
	}

	if err := os.Unsetenv("ALL_CLI_NO_PROGRESS"); err != nil {
		t.Fatal(err)
	}
	if allCliNoProgressEnvSet() {
		t.Fatal("expected ALL_CLI_NO_PROGRESS unset")
	}
	for _, v := range []string{"1", "true", "yes", "on", "TRUE"} {
		if err := os.Setenv("ALL_CLI_NO_PROGRESS", v); err != nil {
			t.Fatal(err)
		}
		if !allCliNoProgressEnvSet() {
			t.Fatalf("expected ALL_CLI_NO_PROGRESS=%q to be set", v)
		}
	}
}

func restoreEnv(t *testing.T, key, value string) {
	t.Helper()
	if value == "" {
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
		return
	}
	if err := os.Setenv(key, value); err != nil {
		t.Fatal(err)
	}
}

func TestSpinnerEnabledTruthTable(t *testing.T) {
	t.Parallel()
	cases := []struct {
		ansi, ci, noProgress, want bool
	}{
		{false, false, false, false},
		{false, false, true, false},
		{false, true, false, false},
		{false, true, true, false},
		{true, false, false, true},
		{true, false, true, false},
		{true, true, false, false},
		{true, true, true, false},
	}
	for _, tc := range cases {
		if got := spinnerEnabled(tc.ansi, tc.ci, tc.noProgress); got != tc.want {
			t.Fatalf("spinnerEnabled(%v,%v,%v)=%v want %v", tc.ansi, tc.ci, tc.noProgress, got, tc.want)
		}
	}
}

// TestTerminalAnsiEnabledCharDevice drives the TTY-true path with /dev/null,
// which is a real character device on unix CI.
func TestTerminalAnsiEnabledCharDevice(t *testing.T) {
	f, err := os.Open(os.DevNull)
	if err != nil {
		t.Skip("no char device")
	}
	defer f.Close()

	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm")
	if !terminalAnsiEnabled(f) {
		t.Fatal("expected ANSI enabled on char device")
	}
	t.Setenv("NO_COLOR", "1")
	if terminalAnsiEnabled(f) {
		t.Fatal("NO_COLOR should disable ANSI")
	}
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "dumb")
	if terminalAnsiEnabled(f) {
		t.Fatal("TERM=dumb should disable ANSI")
	}
}

// TestRainbowAndDimOnTTY swaps os.Stdout for a char device to reach the ANSI
// branches of rainbowLine/dimIfTTY.
func TestRainbowAndDimOnTTY(t *testing.T) {
	f, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Skip("no char device")
	}
	defer f.Close()
	old := os.Stdout
	os.Stdout = f
	t.Cleanup(func() { os.Stdout = old })
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm")

	if got := rainbowLine("hi"); !strings.Contains(got, "\033[") {
		t.Fatalf("rainbowLine on TTY = %q, want ANSI", got)
	}
	if got := dimIfTTY("hi"); got != "\033[2mhi\033[0m" {
		t.Fatalf("dimIfTTY on TTY = %q", got)
	}
}
