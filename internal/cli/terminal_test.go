package cli

import (
	"os"
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
