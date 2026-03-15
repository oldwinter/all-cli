package cli

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
)

func TestKubectlStatusJSON(t *testing.T) {
	summary := model.ToolSummary{
		ID:              "kubectl",
		DisplayName:     "kubectl",
		Category:        "k8s",
		Installed:       true,
		InstallPath:     "/usr/bin/kubectl",
		ConfiguredState: model.ConfiguredYes,
		Configured:      true,
		Capabilities:    model.Capability{HasContexts: true, CanSwitch: true},
		Current:         map[string]string{"context": "prod", "namespace": "payments"},
	}
	stubToolEvaluation(t, "kubectl", summary)

	opts := &rootOptions{JSON: true, Timeout: time.Second}
	stdout, stderr, err := executeTestCommand(t, newKubectlCommand(opts, cliFakeRunner{}), "status")
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
	if got.ID != "kubectl" || got.Current["namespace"] != "payments" {
		t.Fatalf("unexpected summary: %#v", got)
	}
}

func TestKubectlStatusPlain(t *testing.T) {
	summary := model.ToolSummary{
		ID:              "kubectl",
		DisplayName:     "kubectl",
		Category:        "k8s",
		Installed:       true,
		ConfiguredState: model.ConfiguredNo,
		Warnings:        []string{"namespace not set"},
	}
	stubToolEvaluation(t, "kubectl", summary)

	opts := &rootOptions{Timeout: time.Second}
	stdout, stderr, err := executeTestCommand(t, newKubectlCommand(opts, cliFakeRunner{}), "status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "- kubectl: namespace not set") {
		t.Fatalf("unexpected stdout:\n%s", stdout)
	}
}

func TestKubectlCurrentPlain(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"kubectl config current-context": {Stdout: "prod\n"},
			"kubectl config view --minify --output jsonpath={..namespace}{\"\\n\"}": {
				Stdout: "payments\n",
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newKubectlCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, needle := range []string{"context: prod", "namespace: payments"} {
		if !strings.Contains(stdout, needle) {
			t.Fatalf("expected %q in stdout, got:\n%s", needle, stdout)
		}
	}
}

func TestKubectlCurrentJSON(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"kubectl config current-context": {Stdout: "prod\n"},
			"kubectl config view --minify --output jsonpath={..namespace}{\"\\n\"}": {
				Stdout: "payments\n",
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newKubectlCommand(opts, runner), "current")
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
	if got.Current["context"] != "prod" || got.Current["namespace"] != "payments" {
		t.Fatalf("unexpected current: %#v", got.Current)
	}
}

func TestKubectlCurrentPlainError(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"kubectl config current-context": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "context missing",
			},
			"kubectl config view --minify --output jsonpath={..namespace}{\"\\n\"}": {
				Stdout: "payments\n",
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newKubectlCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "namespace: payments") {
		t.Fatalf("unexpected stdout:\n%s", stdout)
	}
	if !strings.Contains(stderr, "error: kubectl config current-context failed (exit=1): context missing") {
		t.Fatalf("expected stderr error, got %q", stderr)
	}
}

func TestKubectlCurrentPlainWithoutNamespace(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"kubectl config current-context": {Stdout: "prod\n"},
			"kubectl config view --minify --output jsonpath={..namespace}{\"\\n\"}": {
				Stdout: "",
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newKubectlCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if strings.Contains(stdout, "namespace:") || !strings.Contains(stdout, "context: prod") {
		t.Fatalf("unexpected stdout:\n%s", stdout)
	}
}

func TestKubectlCurrentJSONError(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"kubectl config current-context": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "context missing",
			},
			"kubectl config view --minify --output jsonpath={..namespace}{\"\\n\"}": {
				Stdout: "payments\n",
			},
		},
	}

	stdout, _, err := executeTestCommand(t, newKubectlCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got struct {
		Current map[string]string `json:"current"`
		Errors  []string          `json:"errors"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if got.Current["namespace"] != "payments" || len(got.Errors) != 1 {
		t.Fatalf("unexpected payload: %#v", got)
	}
}

func TestKubectlCurrentJSONWithoutNamespace(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"kubectl config current-context": {Stdout: "prod\n"},
			"kubectl config view --minify --output jsonpath={..namespace}{\"\\n\"}": {
				Stdout: "",
			},
		},
	}

	stdout, _, err := executeTestCommand(t, newKubectlCommand(opts, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got struct {
		Current map[string]string `json:"current"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if got.Current["context"] != "prod" {
		t.Fatalf("unexpected payload: %#v", got)
	}
	if _, ok := got.Current["namespace"]; ok {
		t.Fatalf("did not expect namespace field, got %#v", got)
	}
}

func TestKubectlListPlainError(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"kubectl config get-contexts -o name": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "list failed",
			},
			"kubectl config current-context": {Stdout: "prod\n"},
			"kubectl config view --minify --output jsonpath={..namespace}{\"\\n\"}": {
				Stdout: "payments\n",
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newKubectlCommand(opts, runner), "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "error: list failed") {
		t.Fatalf("expected stderr error, got %q", stderr)
	}
}

func TestKubectlListJSON(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"kubectl config get-contexts -o name": {Stdout: "prod\ndev\n"},
			"kubectl config current-context":      {Stdout: "prod\n"},
			"kubectl config view --minify --output jsonpath={..namespace}{\"\\n\"}": {
				Stdout: "payments\n",
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newKubectlCommand(opts, runner), "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var got struct {
		Current  map[string]string `json:"current"`
		Contexts []map[string]any  `json:"contexts"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if got.Current["context"] != "prod" || len(got.Contexts) != 2 {
		t.Fatalf("unexpected payload: %#v", got)
	}
	if got.Contexts[0]["name"] != "prod" || got.Contexts[0]["namespace"] != "payments" || got.Contexts[0]["is_current"] != true {
		t.Fatalf("unexpected current context item: %#v", got.Contexts[0])
	}
}

func TestKubectlListPlain(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"kubectl config get-contexts -o name": {Stdout: "prod\ndev\n"},
			"kubectl config current-context":      {Stdout: "prod\n"},
			"kubectl config view --minify --output jsonpath={..namespace}{\"\\n\"}": {
				Stdout: "payments\n",
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newKubectlCommand(opts, runner), "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, needle := range []string{"* prod (namespace=payments)", "  dev"} {
		if !strings.Contains(stdout, needle) {
			t.Fatalf("expected %q in stdout, got:\n%s", needle, stdout)
		}
	}
}

func TestKubectlUsePlainWithNamespace(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"kubectl config use-context prod":                      {},
			"kubectl config set-context prod --namespace payments": {},
			"kubectl config current-context":                       {Stdout: "prod\n"},
			"kubectl config view --minify --output jsonpath={..namespace}{\"\\n\"}": {
				Stdout: "payments\n",
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newKubectlCommand(opts, runner), "use", "prod", "--namespace", "payments")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, needle := range []string{"switched kubectl context to prod", "set namespace to payments"} {
		if !strings.Contains(stdout, needle) {
			t.Fatalf("expected %q in stdout, got:\n%s", needle, stdout)
		}
	}
}

func TestKubectlUseJSONSuccessWithoutNamespace(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"kubectl config use-context prod": {Stdout: "Switched\n"},
			"kubectl config current-context":  {Stdout: "prod\n"},
			"kubectl config view --minify --output jsonpath={..namespace}{\"\\n\"}": {
				Stdout: "payments\n",
			},
		},
	}

	stdout, _, err := executeTestCommand(t, newKubectlCommand(opts, runner), "use", "prod")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got model.UseResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if !got.OK || got.Current["context"] != "prod" {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestKubectlUseJSONFailureOnContextSwitch(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"kubectl config use-context prod": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "context switch failed",
			},
		},
	}

	stdout, _, err := executeTestCommand(t, newKubectlCommand(opts, runner), "use", "prod")
	if err == nil {
		t.Fatal("expected error")
	}
	var got model.UseResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if got.OK || !strings.Contains(got.Error, "context switch failed") {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestKubectlUseJSONFailureOnNamespaceSet(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"kubectl config use-context prod": {},
			"kubectl config set-context prod --namespace payments": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "write failed",
			},
		},
	}

	stdout, _, err := executeTestCommand(t, newKubectlCommand(opts, runner), "use", "prod", "--namespace", "payments")
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

func TestKubectlNamespaceJSONSuccess(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"kubectl config set-context --current --namespace payments": {},
			"kubectl config current-context":                            {Stdout: "prod\n"},
			"kubectl config view --minify --output jsonpath={..namespace}{\"\\n\"}": {
				Stdout: "payments\n",
			},
		},
	}

	stdout, _, err := executeTestCommand(t, newKubectlCommand(opts, runner), "namespace", "payments")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got model.UseResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if !got.OK || got.Current["namespace"] != "payments" {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestKubectlNamespacePlain(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"kubectl config set-context --current --namespace payments": {},
			"kubectl config current-context":                            {Stdout: "prod\n"},
			"kubectl config view --minify --output jsonpath={..namespace}{\"\\n\"}": {
				Stdout: "payments\n",
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newKubectlCommand(opts, runner), "namespace", "payments")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "set current kubectl namespace to payments") {
		t.Fatalf("unexpected stdout:\n%s", stdout)
	}
}

func TestKubectlNamespaceJSONFailure(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"kubectl config set-context --current --namespace payments": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "write failed",
			},
		},
	}

	stdout, _, err := executeTestCommand(t, newKubectlCommand(opts, runner), "namespace", "payments")
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

func TestKubectlListJSONWithoutNamespace(t *testing.T) {
	opts := &rootOptions{JSON: true, Timeout: time.Second}
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"kubectl config get-contexts -o name": {Stdout: "prod\ndev\n"},
			"kubectl config current-context":      {Stdout: "prod\n"},
			"kubectl config view --minify --output jsonpath={..namespace}{\"\\n\"}": {
				Stdout: "",
			},
		},
	}

	stdout, _, err := executeTestCommand(t, newKubectlCommand(opts, runner), "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got struct {
		Contexts []map[string]any `json:"contexts"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if _, ok := got.Contexts[0]["namespace"]; ok {
		t.Fatalf("did not expect namespace field, got %#v", got.Contexts[0])
	}
}
