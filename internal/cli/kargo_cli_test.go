package cli

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
)

const kargoConfigOutput = `apiAddress: https://kargo.example.com
defaultProject: payments
`

func TestKargoCurrentJSON(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"kargo config view": {
				Stdout: kargoConfigOutput,
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newKargoCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	var got struct {
		Current map[string]string `json:"current"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if got.Current["api_address"] != "https://kargo.example.com" || got.Current["project"] != "payments" {
		t.Fatalf("unexpected current: %#v", got.Current)
	}
}

func TestKargoStatusPlain(t *testing.T) {
	summary := model.ToolSummary{
		ID:              "kargo",
		DisplayName:     "kargo",
		Category:        "cicd",
		Installed:       true,
		ConfiguredState: model.ConfiguredYes,
		Configured:      true,
		Warnings:        []string{"project drift"},
	}
	stubToolEvaluation(t, "kargo", summary)

	opts := &rootOptions{Timeout: time.Second}
	stdout, stderr, err := executeTestCommand(t, newKargoCommand(opts, cliFakeRunner{}), "status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "- kargo: project drift") {
		t.Fatalf("unexpected stdout:\n%s", stdout)
	}
}

func TestKargoStatusJSON(t *testing.T) {
	summary := model.ToolSummary{
		ID:              "kargo",
		DisplayName:     "kargo",
		Category:        "cicd",
		Installed:       true,
		ConfiguredState: model.ConfiguredYes,
		Configured:      true,
		Current:         map[string]string{"project": "payments"},
	}
	stubToolEvaluation(t, "kargo", summary)

	opts := &rootOptions{JSON: true, Timeout: time.Second}
	stdout, stderr, err := executeTestCommand(t, newKargoCommand(opts, cliFakeRunner{}), "status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	var got model.ToolSummary
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if got.ID != "kargo" || got.Current["project"] != "payments" {
		t.Fatalf("unexpected summary: %#v", got)
	}
}

func TestKargoCurrentPlain(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"kargo config view": {
				Stdout: kargoConfigOutput,
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newKargoCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, needle := range []string{"api_address: https://kargo.example.com", "project: payments"} {
		if !strings.Contains(stdout, needle) {
			t.Fatalf("expected %q in stdout, got:\n%s", needle, stdout)
		}
	}
}

func TestKargoCurrentPlainError(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"kargo config view": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "config unreadable",
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newKargoCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "error: config unreadable") {
		t.Fatalf("expected stderr error, got %q", stderr)
	}
}

func TestKargoCurrentJSONError(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"kargo config view": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "config unreadable",
			},
		},
	}

	stdout, _, err := executeTestCommand(t, newKargoCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got struct {
		Errors []string `json:"errors"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if len(got.Errors) != 2 || got.Errors[0] != "config unreadable" || !strings.Contains(got.Errors[1], "kargo config view failed") {
		t.Fatalf("unexpected payload: %#v", got)
	}
}

func TestKargoUseJSONSuccess(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"kargo config set-project payments": {},
			"kargo config view": {
				Stdout: kargoConfigOutput,
			},
		},
	}

	stdout, _, err := executeTestCommand(t, newKargoCommand(opts, runner), "use", "payments")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got model.UseResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if !got.OK || got.ToolID != "kargo" || got.Current["project"] != "payments" {
		t.Fatalf("unexpected use result: %#v", got)
	}
}

func TestKargoUseUnsetPlain(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"kargo config set-project ": {},
			"kargo config view": {
				Stdout: "apiAddress: https://kargo.example.com\n",
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newKargoCommand(opts, runner), "use", "--unset")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "unset kargo default project") {
		t.Fatalf("unexpected stdout:\n%s", stdout)
	}
}

func TestKargoUsePlainSetProject(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"kargo config set-project payments": {},
			"kargo config view": {
				Stdout: kargoConfigOutput,
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newKargoCommand(opts, runner), "use", "payments")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "set kargo default project to payments") {
		t.Fatalf("unexpected stdout:\n%s", stdout)
	}
}

func TestKargoUseRejectsUnsetWithProject(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	stdout, _, err := executeTestCommand(t, newKargoCommand(opts, cliFakeRunner{}), "use", "payments", "--unset")
	if err == nil {
		t.Fatal("expected error")
	}
	if stdout != "" {
		t.Fatalf("expected no stdout, got %q", stdout)
	}
	if !strings.Contains(err.Error(), "use either --unset or a project name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestKargoUseJSONFailure(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"kargo config set-project payments": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "write failed",
			},
		},
	}

	stdout, _, err := executeTestCommand(t, newKargoCommand(opts, runner), "use", "payments")
	if err == nil {
		t.Fatal("expected error")
	}
	var got model.UseResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if got.OK || !strings.Contains(got.Error, "write failed") {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestKargoUseErrorsWhenProjectMissing(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	_, _, err := executeTestCommand(t, newKargoCommand(opts, cliFakeRunner{}), "use")
	if err == nil || !strings.Contains(err.Error(), "project name is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}
