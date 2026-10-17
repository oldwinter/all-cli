package cli

import (
	"testing"
)

// The tests in this file mutate process-global environment variables, and the
// code under test reads them through os.Getenv. They therefore must not call
// t.Parallel(): a parallel test that consults the same variables (for example
// the options command reporting no_progress, or status deciding whether to draw
// a spinner) would observe a value written by another test's goroutine.
//
// That is not theoretical: TestOptionsCommandOutputsPersistentFlagsAsText
// failed the three-pass stability lane with "expected no_progress=false" while
// TestStatusSpinnerEnvHelpers had ALL_CLI_NO_PROGRESS set concurrently.
// t.Setenv both isolates the value and restores it automatically, and it panics
// if the test is marked parallel, which keeps this constraint enforced.

func TestAnsiDisabledByEnv(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "")

	if ansiDisabledByEnv() {
		t.Fatal("expected ansi not disabled with empty env")
	}

	t.Setenv("NO_COLOR", "1")
	if !ansiDisabledByEnv() {
		t.Fatal("expected NO_COLOR to disable ansi")
	}

	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "dumb")
	if !ansiDisabledByEnv() {
		t.Fatal("expected TERM=dumb to disable ansi")
	}
}

func TestStatusSpinnerEnvHelpers(t *testing.T) {
	t.Setenv("CI", "")
	t.Setenv("ALL_CLI_NO_PROGRESS", "")

	if ciEnvSet() {
		t.Fatal("expected CI unset to be false")
	}

	t.Setenv("CI", "true")
	if !ciEnvSet() {
		t.Fatal("expected CI set")
	}

	t.Setenv("ALL_CLI_NO_PROGRESS", "")
	if allCliNoProgressEnvSet() {
		t.Fatal("expected ALL_CLI_NO_PROGRESS unset")
	}
	for _, v := range []string{"1", "true", "yes", "on", "TRUE"} {
		t.Setenv("ALL_CLI_NO_PROGRESS", v)
		if !allCliNoProgressEnvSet() {
			t.Fatalf("expected ALL_CLI_NO_PROGRESS=%q to be set", v)
		}
	}
}
