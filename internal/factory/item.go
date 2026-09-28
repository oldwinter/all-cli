// Package factory implements a small repository-local work pipeline:
// work items live as JSON files in .factory/backlog and move through an
// ordered state machine with recorded evidence.
package factory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ItemSchemaVersion is the version of .factory/work-item.schema.json that
// backlog files declare in schema_version.
const ItemSchemaVersion = "factory-item-v0.1"

// State is one ordered step in the factory pipeline.
type State string

const (
	StateQueued     State = "queued"
	StateInProgress State = "in_progress"
	StateVerifying  State = "verifying"
	StateVerified   State = "verified"
	StateDelivered  State = "delivered"
	StateFailed     State = "failed"
	StateBlocked    State = "blocked"
)

// States lists every state in pipeline order (failed/blocked are lateral).
func States() []State {
	return []State{
		StateQueued,
		StateInProgress,
		StateVerifying,
		StateVerified,
		StateDelivered,
		StateFailed,
		StateBlocked,
	}
}

var transitions = map[State][]State{
	StateQueued:     {StateInProgress, StateBlocked},
	StateInProgress: {StateVerifying, StateFailed, StateBlocked, StateQueued},
	StateVerifying:  {StateVerified, StateFailed, StateBlocked},
	StateVerified:   {StateDelivered, StateBlocked, StateVerifying, StateInProgress},
	StateFailed:     {StateInProgress, StateQueued, StateBlocked, StateVerifying},
	StateBlocked:    {StateQueued},
	StateDelivered:  {},
}

// CanTransition reports whether moving from one state to another is allowed.
func CanTransition(from, to State) bool {
	for _, next := range transitions[from] {
		if next == to {
			return true
		}
	}
	return false
}

// Terminal reports whether a state ends the pipeline (delivered).
func (s State) Terminal() bool { return s == StateDelivered }

// Evidence is one recorded event in a work item's history.
type Evidence struct {
	TS       string `json:"ts"`
	Event    string `json:"event"`
	Command  string `json:"command,omitempty"`
	ExitCode int    `json:"exit_code,omitempty"`
	Note     string `json:"note,omitempty"`
	Head     string `json:"head,omitempty"`
	Log      string `json:"log,omitempty"`
}

// Source links a work item back to its canonical queue entry.
type Source struct {
	Issue    int   `json:"issue,omitempty"`
	AvoidPRs []int `json:"avoid_prs,omitempty"`
}

// Verified binds a passing verification run to the exact acceptance criteria
// and checks that ran. Delivering requires the fingerprint to still match.
type Verified struct {
	At          string `json:"at"`
	Head        string `json:"head,omitempty"`
	Fingerprint string `json:"fingerprint"`
}

// WorkItem is one unit of factory work with acceptance criteria and checks.
type WorkItem struct {
	SchemaVersion string     `json:"schema_version"`
	ID            string     `json:"id"`
	Title         string     `json:"title"`
	Kind          string     `json:"kind"`
	State         State      `json:"state"`
	Order         int        `json:"order"`
	Source        Source     `json:"source,omitempty"`
	Acceptance    []string   `json:"acceptance"`
	Checks        []string   `json:"checks"`
	Branch        string     `json:"branch,omitempty"`
	Attempts      int        `json:"attempts"`
	LastError     string     `json:"last_error,omitempty"`
	Verified      *Verified  `json:"verified,omitempty"`
	CreatedAt     string     `json:"created_at"`
	UpdatedAt     string     `json:"updated_at"`
	Evidence      []Evidence `json:"evidence,omitempty"`
}

var idPattern = regexp.MustCompile(`^[A-Z][A-Z0-9]*-[0-9]+$`)

var kinds = map[string]bool{
	"improvement": true,
	"bug":         true,
	"test":        true,
	"docs":        true,
}

// Validate checks the fields the schema cannot fully express or that callers
// rely on before trusting an item.
func (it *WorkItem) Validate() error {
	if it.SchemaVersion != ItemSchemaVersion {
		return fmt.Errorf("item %q: schema_version %q, want %q", it.ID, it.SchemaVersion, ItemSchemaVersion)
	}
	if !idPattern.MatchString(it.ID) {
		return fmt.Errorf("item id %q must match %s", it.ID, idPattern.String())
	}
	if strings.TrimSpace(it.Title) == "" {
		return fmt.Errorf("item %s: title is required", it.ID)
	}
	if !kinds[it.Kind] {
		return fmt.Errorf("item %s: kind %q, want one of improvement|bug|test|docs", it.ID, it.Kind)
	}
	valid := false
	for _, s := range States() {
		if it.State == s {
			valid = true
		}
	}
	if !valid {
		return fmt.Errorf("item %s: unknown state %q", it.ID, it.State)
	}
	if len(it.Acceptance) == 0 {
		return fmt.Errorf("item %s: at least one acceptance criterion is required", it.ID)
	}
	if len(it.Checks) == 0 {
		return fmt.Errorf("item %s: at least one verification check is required", it.ID)
	}
	for i, c := range it.Checks {
		if strings.TrimSpace(c) == "" {
			return fmt.Errorf("item %s: check %d is empty", it.ID, i)
		}
	}
	return nil
}

// AcceptanceFingerprint hashes the ordered acceptance criteria and checks.
// Any edit to either invalidates a prior verification.
func (it *WorkItem) AcceptanceFingerprint() string {
	payload, _ := json.Marshal(struct {
		Acceptance []string `json:"acceptance"`
		Checks     []string `json:"checks"`
	}{Acceptance: it.Acceptance, Checks: it.Checks})
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

// Transition moves the item to a new state when allowed and stamps UpdatedAt.
func (it *WorkItem) Transition(to State, now time.Time) error {
	if it.State == to {
		return nil
	}
	if !CanTransition(it.State, to) {
		return fmt.Errorf("item %s: cannot move %s -> %s", it.ID, it.State, to)
	}
	it.State = to
	it.UpdatedAt = now.UTC().Format(time.RFC3339)
	return nil
}

// Record appends an evidence entry and stamps UpdatedAt.
func (it *WorkItem) Record(ev Evidence, now time.Time) {
	if ev.TS == "" {
		ev.TS = now.UTC().Format(time.RFC3339)
	}
	it.Evidence = append(it.Evidence, ev)
	it.UpdatedAt = now.UTC().Format(time.RFC3339)
}
