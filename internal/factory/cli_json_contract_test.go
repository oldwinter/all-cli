package factory

import (
	"encoding/json"
	"strings"
	"testing"
)

// JSON consumers must be able to rely on stable output types: an empty
// backlog is "[]", an empty queue item is "null" — never bare text.
func TestListJSONEmptyBacklogEmitsArray(t *testing.T) {
	opts, _ := testOptions(t, &fakeExec{})

	stdout, _, err := run(t, opts, "list", "--json")
	if err != nil {
		t.Fatalf("list --json: %v", err)
	}
	var items []map[string]any
	if err := json.Unmarshal([]byte(stdout), &items); err != nil {
		t.Fatalf("list --json output is not a JSON array: %q", stdout)
	}
	if items == nil || len(items) != 0 {
		t.Fatalf("expected empty array, got %q", stdout)
	}
}

func TestListJSONStateFilterMissEmitsArray(t *testing.T) {
	opts, _ := testOptions(t, &fakeExec{})
	intakeOK(t, opts, "WI-301") // queued

	stdout, _, err := run(t, opts, "list", "--state", "delivered", "--json")
	if err != nil {
		t.Fatalf("list --state delivered --json: %v", err)
	}
	var items []map[string]any
	if err := json.Unmarshal([]byte(stdout), &items); err != nil {
		t.Fatalf("filtered list --json output is not a JSON array: %q", stdout)
	}
	if items == nil || len(items) != 0 {
		t.Fatalf("expected empty array for filter miss, got %q", stdout)
	}
}

func TestStatusJSONEmptyBacklogEmitsEmptyItems(t *testing.T) {
	opts, _ := testOptions(t, &fakeExec{})

	stdout, _, err := run(t, opts, "status", "--json")
	if err != nil {
		t.Fatalf("status --json: %v", err)
	}
	var payload struct {
		Total  int              `json:"total"`
		Counts map[string]int   `json:"counts"`
		Items  []map[string]any `json:"items"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("status --json output is not an object: %q", stdout)
	}
	if payload.Total != 0 || payload.Items == nil || len(payload.Items) != 0 {
		t.Fatalf("expected total=0 with empty items array, got %q", stdout)
	}
}

func TestNextJSONEmptyQueueEmitsNull(t *testing.T) {
	opts, _ := testOptions(t, &fakeExec{})

	stdout, _, err := run(t, opts, "next", "--json")
	if err != nil {
		t.Fatalf("next --json: %v", err)
	}
	var v any
	if err := json.Unmarshal([]byte(stdout), &v); err != nil {
		t.Fatalf("next --json output is not valid JSON: %q", stdout)
	}
	if v != nil {
		t.Fatalf("expected null for empty queue, got %q", stdout)
	}
	if strings.Contains(stdout, "no queued items") {
		t.Fatalf("expected JSON null, got human text: %q", stdout)
	}
}

func TestNextJSONQueuedItemEmitsObject(t *testing.T) {
	opts, _ := testOptions(t, &fakeExec{})
	intakeOK(t, opts, "WI-302")

	stdout, _, err := run(t, opts, "next", "--json")
	if err != nil {
		t.Fatalf("next --json: %v", err)
	}
	var item map[string]any
	if err := json.Unmarshal([]byte(stdout), &item); err != nil {
		t.Fatalf("next --json output is not an object: %q", stdout)
	}
	if item["id"] != "WI-302" {
		t.Fatalf("next --json id = %v, want WI-302", item["id"])
	}
}
