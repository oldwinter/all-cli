package factory

import (
	"strings"
	"testing"
	"time"
)

func validItem() *WorkItem {
	return &WorkItem{
		SchemaVersion: ItemSchemaVersion,
		ID:            "WI-001",
		Title:         "example",
		Kind:          "improvement",
		State:         StateQueued,
		Order:         10,
		Acceptance:    []string{"works"},
		Checks:        []string{"true"},
	}
}

func TestWorkItemValidate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		mutate  func(*WorkItem)
		wantErr string
	}{
		{"valid", func(*WorkItem) {}, ""},
		{"schema version", func(it *WorkItem) { it.SchemaVersion = "nope" }, "schema_version"},
		{"empty schema version", func(it *WorkItem) { it.SchemaVersion = "" }, "schema_version"},
		{"bad id", func(it *WorkItem) { it.ID = "wi-1" }, "id"},
		{"empty title", func(it *WorkItem) { it.Title = "  " }, "title"},
		{"bad kind", func(it *WorkItem) { it.Kind = "epic" }, "kind"},
		{"bad state", func(it *WorkItem) { it.State = "limbo" }, "state"},
		{"no acceptance", func(it *WorkItem) { it.Acceptance = nil }, "acceptance"},
		{"no checks", func(it *WorkItem) { it.Checks = nil }, "check"},
		{"empty check", func(it *WorkItem) { it.Checks = []string{" "} }, "check"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			item := validItem()
			tc.mutate(item)
			err := item.Validate()
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Validate() = %v, want error containing %q", err, tc.wantErr)
			}
		})
	}
}

func TestTransitions(t *testing.T) {
	t.Parallel()

	legal := []struct{ from, to State }{
		{StateQueued, StateInProgress},
		{StateQueued, StateBlocked},
		{StateInProgress, StateVerifying},
		{StateInProgress, StateFailed},
		{StateInProgress, StateBlocked},
		{StateInProgress, StateQueued},
		{StateVerifying, StateVerified},
		{StateVerifying, StateFailed},
		{StateVerifying, StateBlocked},
		{StateVerified, StateDelivered},
		{StateVerified, StateBlocked},
		{StateVerified, StateVerifying},
		{StateVerified, StateInProgress},
		{StateFailed, StateInProgress},
		{StateFailed, StateQueued},
		{StateFailed, StateBlocked},
		{StateFailed, StateVerifying},
		{StateBlocked, StateQueued},
	}
	for _, tc := range legal {
		if !CanTransition(tc.from, tc.to) {
			t.Errorf("CanTransition(%s, %s) = false, want true", tc.from, tc.to)
		}
	}

	illegal := []struct{ from, to State }{
		{StateQueued, StateVerified},
		{StateQueued, StateDelivered},
		{StateQueued, StateVerifying},
		{StateDelivered, StateInProgress},
		{StateDelivered, StateFailed},
		{StateDelivered, StateQueued},
		{StateBlocked, StateDelivered},
		{StateBlocked, StateVerifying},
	}
	for _, tc := range illegal {
		if CanTransition(tc.from, tc.to) {
			t.Errorf("CanTransition(%s, %s) = true, want false", tc.from, tc.to)
		}
	}
}

func TestTransitionStampsUpdatedAt(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	item := validItem()
	if err := item.Transition(StateInProgress, now); err != nil {
		t.Fatal(err)
	}
	if item.State != StateInProgress {
		t.Fatalf("state = %s", item.State)
	}
	if item.UpdatedAt != "2026-09-28T12:00:00Z" {
		t.Fatalf("updated_at = %s", item.UpdatedAt)
	}
	if err := item.Transition(StateInProgress, now); err != nil {
		t.Fatalf("same-state transition should be a no-op: %v", err)
	}
	if err := item.Transition(StateDelivered, now); err == nil {
		t.Fatal("expected illegal transition error")
	}
}

func TestRecordDefaultsTimestamp(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	item := validItem()
	item.Record(Evidence{Event: "note", Note: "hello"}, now)
	if len(item.Evidence) != 1 {
		t.Fatalf("evidence len = %d", len(item.Evidence))
	}
	if item.Evidence[0].TS != "2026-09-28T12:00:00Z" {
		t.Fatalf("ts = %s", item.Evidence[0].TS)
	}
	item.Record(Evidence{Event: "note", TS: "2020-01-01T00:00:00Z"}, now)
	if item.Evidence[1].TS != "2020-01-01T00:00:00Z" {
		t.Fatalf("explicit ts overwritten: %s", item.Evidence[1].TS)
	}
}

func TestTerminalState(t *testing.T) {
	t.Parallel()

	if !StateDelivered.Terminal() {
		t.Fatal("delivered should be terminal")
	}
	if StateFailed.Terminal() || StateQueued.Terminal() {
		t.Fatal("non-delivered states must not be terminal")
	}
}
