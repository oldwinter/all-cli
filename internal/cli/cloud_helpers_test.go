package cli

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/oldwinter/all-cli/internal/model"
)

func TestNonNilNormalizesNilSlice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   []string
		want int
	}{
		{name: "nil", in: nil, want: 0},
		{name: "empty", in: []string{}, want: 0},
		{name: "one", in: []string{"a"}, want: 1},
		{name: "two", in: []string{"a", "b"}, want: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := nonNil(tt.in)
			if got == nil {
				t.Fatalf("nonNil(%#v) returned nil, want non-nil slice", tt.in)
			}
			if len(got) != tt.want {
				t.Fatalf("len(nonNil(%#v)) = %d, want %d", tt.in, len(got), tt.want)
			}
		})
	}
}

// TestCurrentEnvelopeMarshalsEmptySlicesAsArrays pins the wire contract that
// motivated the typed envelope: a consumer must never need a null branch.
func TestCurrentEnvelopeMarshalsEmptySlicesAsArrays(t *testing.T) {
	t.Parallel()

	encoded, err := json.Marshal(newCurrentEnvelope(map[string]string{"context": "prod"}, nil, nil))
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	got := string(encoded)

	for _, needle := range []string{`"warnings":[]`, `"errors":[]`} {
		if !strings.Contains(got, needle) {
			t.Fatalf("expected %s in %s", needle, got)
		}
	}
	if strings.Contains(got, "null") {
		t.Fatalf("envelope must not contain null, got %s", got)
	}
}

func TestCurrentEnvelopePreservesValues(t *testing.T) {
	t.Parallel()

	env := newCurrentEnvelope(
		map[string]string{"context": "prod", "namespace": "payments"},
		[]string{"warning one"},
		[]string{"error one"},
	)

	if env.Current["context"] != "prod" || env.Current["namespace"] != "payments" {
		t.Fatalf("unexpected current: %#v", env.Current)
	}
	if len(env.Warnings) != 1 || env.Warnings[0] != "warning one" {
		t.Fatalf("unexpected warnings: %#v", env.Warnings)
	}
	if len(env.Errors) != 1 || env.Errors[0] != "error one" {
		t.Fatalf("unexpected errors: %#v", env.Errors)
	}
}

// TestCurrentEnvelopeDedupesMessages covers adapters that report the same
// underlying failure from two probes: the payload must not repeat it.
func TestCurrentEnvelopeDedupesMessages(t *testing.T) {
	t.Parallel()

	env := newCurrentEnvelope(
		nil,
		[]string{"same warning", " same warning ", ""},
		[]string{"same error", "same error"},
	)

	if len(env.Warnings) != 1 || env.Warnings[0] != "same warning" {
		t.Fatalf("warnings not deduped: %#v", env.Warnings)
	}
	if len(env.Errors) != 1 || env.Errors[0] != "same error" {
		t.Fatalf("errors not deduped: %#v", env.Errors)
	}
}

func TestCurrentEnvelopeJSONRoundTripsIntoStatusShape(t *testing.T) {
	t.Parallel()

	encoded, err := json.Marshal(newCurrentEnvelope(map[string]string{"host": "github.com"}, nil, []string{"boom"}))
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}

	var decoded struct {
		Current  map[string]string `json:"current"`
		Warnings []string          `json:"warnings"`
		Errors   []string          `json:"errors"`
	}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if decoded.Current["host"] != "github.com" {
		t.Fatalf("unexpected current: %#v", decoded.Current)
	}
	if decoded.Warnings == nil || len(decoded.Warnings) != 0 {
		t.Fatalf("warnings should decode to an empty non-nil slice, got %#v", decoded.Warnings)
	}
	if len(decoded.Errors) != 1 || decoded.Errors[0] != "boom" {
		t.Fatalf("unexpected errors: %#v", decoded.Errors)
	}
}

// TestReportUseFailureJSONPrintsEnvelopeOnce is the P1-8 contract: with --json a
// failed `use` writes the failure envelope to stdout exactly once and suppresses
// the duplicate "Error: ..." line on stderr.
func TestReportUseFailureJSONPrintsEnvelopeOnce(t *testing.T) {
	t.Parallel()

	root := NewRootCommand()
	var stdout strings.Builder
	var stderr strings.Builder
	root.SetOut(&stdout)
	root.SetErr(&stderr)

	opts := &rootOptions{JSON: true, Timeout: time.Second}
	wantErr := assertError("context switch failed")

	got := reportUseFailure(root, opts, "docker", wantErr)
	if got == nil || got.Error() != wantErr.Error() {
		t.Fatalf("reportUseFailure returned %v, want the original error", got)
	}
	if !root.SilenceErrors {
		t.Fatal("expected SilenceErrors to be set so cobra does not repeat the error on stderr")
	}

	var decoded model.UseResult
	if err := json.Unmarshal([]byte(stdout.String()), &decoded); err != nil {
		t.Fatalf("decode use result: %v\n%s", err, stdout.String())
	}
	if decoded.OK {
		t.Fatalf("expected ok=false, got %#v", decoded)
	}
	if decoded.ToolID != "docker" {
		t.Fatalf("tool_id = %q, want docker", decoded.ToolID)
	}
	if !strings.Contains(decoded.Error, "context switch failed") {
		t.Fatalf("unexpected error payload: %q", decoded.Error)
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty (no duplicate report)", stderr.String())
	}
	if count := strings.Count(stdout.String(), "context switch failed"); count != 1 {
		t.Fatalf("failure reported %d times on stdout, want 1:\n%s", count, stdout.String())
	}
}

// TestReportUseFailurePlainLeavesStderrAlone covers the non-JSON path: the helper
// must not print anything, so the single human-readable report comes from cobra.
func TestReportUseFailurePlainLeavesStderrAlone(t *testing.T) {
	t.Parallel()

	root := NewRootCommand()
	var stdout strings.Builder
	var stderr strings.Builder
	root.SetOut(&stdout)
	root.SetErr(&stderr)

	opts := &rootOptions{Timeout: time.Second}
	wantErr := assertError("context switch failed")

	got := reportUseFailure(root, opts, "docker", wantErr)
	if got == nil || got.Error() != wantErr.Error() {
		t.Fatalf("reportUseFailure returned %v, want the original error", got)
	}
	if root.SilenceErrors {
		t.Fatal("plain output should let cobra report the error on stderr")
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty for plain output", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty (cobra owns the report)", stderr.String())
	}
}

// TestSingleToolStatusReportCarriesSchemaMetadata is the P1-5 contract: per-tool
// status JSON must be a full StatusReport so it validates against
// schemas/status-report-v0.1.json and can be piped into `all-cli diff`.
func TestSingleToolStatusReportCarriesSchemaMetadata(t *testing.T) {
	t.Parallel()

	summary := model.ToolSummary{ID: "kubectl", DisplayName: "kubectl", Category: "k8s", Installed: true}
	report := singleToolStatusReport(summary)

	if report.SchemaVersion != model.SchemaVersionV01 {
		t.Fatalf("schema_version = %q, want %q", report.SchemaVersion, model.SchemaVersionV01)
	}
	if report.GeneratedAt.IsZero() {
		t.Fatal("generated_at must be set")
	}
	if report.Legend.Installed == "" {
		t.Fatal("expected legend to be populated")
	}
	if len(report.Tools) != 1 || report.Tools[0].ID != "kubectl" {
		t.Fatalf("unexpected tools: %#v", report.Tools)
	}
}

func TestSingleToolStatusReportIsDiffable(t *testing.T) {
	t.Parallel()

	report := singleToolStatusReport(model.ToolSummary{ID: "kubectl", Category: "k8s", Installed: true})
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}

	var decoded model.StatusReport
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if strings.TrimSpace(decoded.SchemaVersion) == "" {
		t.Fatal("round-tripped report lost schema_version, so `all-cli diff` would reject it")
	}
}

func TestPrintSingleToolStatusRendersTable(t *testing.T) {
	t.Parallel()

	var out strings.Builder
	printSingleToolStatus(&out, model.ToolSummary{
		ID:              "docker",
		DisplayName:     "docker",
		Category:        "containers",
		Installed:       true,
		ConfiguredState: model.ConfiguredYes,
	})

	if !strings.Contains(out.String(), "TOOL") {
		t.Fatalf("expected table header, got:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "docker") {
		t.Fatalf("expected docker row, got:\n%s", out.String())
	}
}

func TestDedupeMessagesKeepsFirstOccurrenceOrder(t *testing.T) {
	t.Parallel()

	got := dedupeMessages([]string{"b", "a", " b ", "", "a"})
	want := []string{"b", "a"}
	if len(got) != len(want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %#v, want %#v", got, want)
		}
	}
}

func TestPrintDiagnosticsDedupesAndPrefixes(t *testing.T) {
	t.Parallel()

	var out strings.Builder
	printDiagnostics(&out, []string{"w", "w"}, []string{"e", "e"})
	got := out.String()

	if strings.Count(got, "warning: w") != 1 {
		t.Fatalf("expected one deduped warning line, got:\n%s", got)
	}
	if strings.Count(got, "error: e") != 1 {
		t.Fatalf("expected one deduped error line, got:\n%s", got)
	}
}
