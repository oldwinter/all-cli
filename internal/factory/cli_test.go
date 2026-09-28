package factory

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
)

type fakeExec struct {
	calls   []string
	results map[string]execx.CmdResult
	def     execx.CmdResult
}

func (f *fakeExec) Run(_ context.Context, name string, args ...string) execx.CmdResult {
	joined := name + " " + strings.Join(args, " ")
	f.calls = append(f.calls, joined)
	for key, res := range f.results {
		if strings.Contains(joined, key) {
			return res
		}
	}
	return f.def
}

func testOptions(t *testing.T, exec execx.Runner) (*options, string) {
	t.Helper()
	return testOptionsWithHead(t, exec, func() string { return "abc1234" })
}

func testOptionsWithHead(t *testing.T, exec execx.Runner, head func() string) (*options, string) {
	t.Helper()
	dir := t.TempDir()
	now := func() time.Time { return time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC) }
	return &options{
		root: dir,
		now:  now,
		head: head,
		exec: Runner{
			Exec:   exec,
			Root:   dir,
			RunDir: filepath.Join(dir, ".factory", "run"),
			Now:    now,
		},
	}, dir
}

func run(t *testing.T, opts *options, args ...string) (string, string, error) {
	t.Helper()
	var out, errOut bytes.Buffer
	cmd := newRootCommand(opts)
	cmd.SetArgs(args)
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	err := cmd.Execute()
	return out.String(), errOut.String(), err
}

func intakeOK(t *testing.T, opts *options, id string) {
	t.Helper()
	stdout, _, err := run(t, opts, "intake",
		"--id", id,
		"--title", "sample "+id,
		"--acceptance", "it works",
		"--check", "check-one",
		"--check", "check-two")
	if err != nil {
		t.Fatalf("intake %s: %v", id, err)
	}
	if !strings.Contains(stdout, "queued "+id) {
		t.Fatalf("intake output = %q", stdout)
	}
}

func TestIntakeDryRunDoesNotWrite(t *testing.T) {
	opts, _ := testOptions(t, &fakeExec{})

	stdout, _, err := run(t, opts, "intake", "--id", "WI-001",
		"--title", "demo", "--acceptance", "a", "--check", "c", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "dry-run") {
		t.Fatalf("dry-run output = %q", stdout)
	}
	if opts.store().Exists("WI-001") {
		t.Fatal("dry-run intake wrote a file")
	}
}

func TestNextPrintsCriteria(t *testing.T) {
	opts, _ := testOptions(t, &fakeExec{})
	intakeOK(t, opts, "WI-001")

	stdout, _, err := run(t, opts, "next")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "WI-001") || !strings.Contains(stdout, "accept: it works") {
		t.Fatalf("next output = %q", stdout)
	}
}

func TestClaimTransitionsAndIdempotence(t *testing.T) {
	opts, _ := testOptions(t, &fakeExec{})
	intakeOK(t, opts, "WI-001")

	stdout, _, err := run(t, opts, "claim", "WI-001")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "in_progress") {
		t.Fatalf("claim output = %q", stdout)
	}
	item, err := opts.store().Load("WI-001")
	if err != nil {
		t.Fatal(err)
	}
	if item.State != StateInProgress || item.Attempts != 1 {
		t.Fatalf("after claim: state=%s attempts=%d", item.State, item.Attempts)
	}
	if item.Branch != "factory/wi-001-sample-wi-001" {
		t.Fatalf("branch = %q", item.Branch)
	}

	stdout, _, err = run(t, opts, "claim", "WI-001")
	if err != nil || !strings.Contains(stdout, "already in_progress") {
		t.Fatalf("idempotent claim: out=%q err=%v", stdout, err)
	}
	item, _ = opts.store().Load("WI-001")
	if item.Attempts != 1 {
		t.Fatalf("idempotent claim changed attempts to %d", item.Attempts)
	}
}

func TestVerifyRecordsEvidence(t *testing.T) {
	exec := &fakeExec{def: execx.CmdResult{Stdout: "ok"}}
	opts, dir := testOptions(t, exec)
	intakeOK(t, opts, "WI-001")
	if _, _, err := run(t, opts, "claim", "WI-001"); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := run(t, opts, "verify", "WI-001")
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !strings.Contains(stdout, "verified: 2/2") {
		t.Fatalf("verify output = %q", stdout)
	}
	for _, want := range []string{"check 1/2 pass: check-one", "check 2/2 pass: check-two"} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("progress output missing %q: %q", want, stderr)
		}
	}
	item, _ := opts.store().Load("WI-001")
	if item.State != StateVerified {
		t.Fatalf("state = %s", item.State)
	}
	var checkEvents int
	for _, ev := range item.Evidence {
		if ev.Event == "check" {
			checkEvents++
			if ev.ExitCode != 0 || ev.Log == "" {
				t.Fatalf("check evidence = %+v", ev)
			}
		}
	}
	if checkEvents != 2 {
		t.Fatalf("check evidence events = %d", checkEvents)
	}
	if _, err := os.Stat(filepath.Join(dir, ".factory", "run", "WI-001")); err != nil {
		t.Fatal("run log dir missing")
	}
}

func TestDeliverAndTerminalNoOps(t *testing.T) {
	opts, _ := testOptions(t, &fakeExec{def: execx.CmdResult{}})
	intakeOK(t, opts, "WI-001")
	if _, _, err := run(t, opts, "claim", "WI-001"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := run(t, opts, "verify", "WI-001"); err != nil {
		t.Fatal(err)
	}

	stdout, _, err := run(t, opts, "deliver", "WI-001", "--note", "branch pushed")
	if err != nil || !strings.Contains(stdout, "delivered WI-001") {
		t.Fatalf("deliver: out=%q err=%v", stdout, err)
	}
	stdout, _, err = run(t, opts, "deliver", "WI-001")
	if err != nil || !strings.Contains(stdout, "already delivered") {
		t.Fatalf("idempotent deliver: out=%q err=%v", stdout, err)
	}
	stdout, _, err = run(t, opts, "verify", "WI-001")
	if err != nil || !strings.Contains(stdout, "no-op") {
		t.Fatalf("verify on delivered: out=%q err=%v", stdout, err)
	}

	stdout, _, err = run(t, opts, "status")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "items=1") || !strings.Contains(stdout, "delivered=1") {
		t.Fatalf("status output = %q", stdout)
	}
}

// TestDeliverRejectsStaleVerification reproduces the coordinator's WI-999
// report: verify passes, the item's checks are then edited on disk, and
// deliver must refuse instead of shipping stale evidence.
func TestDeliverRejectsStaleVerification(t *testing.T) {
	opts, _ := testOptions(t, &fakeExec{def: execx.CmdResult{}})
	intakeOK(t, opts, "WI-999")
	if _, _, err := run(t, opts, "claim", "WI-999"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := run(t, opts, "verify", "WI-999"); err != nil {
		t.Fatal(err)
	}

	tamper := func() *WorkItem {
		item, err := opts.store().Load("WI-999")
		if err != nil {
			t.Fatal(err)
		}
		item.Checks = []string{"exit 1"}
		if err := opts.store().Save(item); err != nil {
			t.Fatal(err)
		}
		return item
	}
	tamper()

	_, _, err := run(t, opts, "deliver", "WI-999")
	if err == nil || !strings.Contains(err.Error(), "changed after verification") {
		t.Fatalf("deliver err = %v, want stale verification rejection", err)
	}
	item, _ := opts.store().Load("WI-999")
	if item.State != StateInProgress {
		t.Fatalf("state = %s, want in_progress after stale rejection", item.State)
	}
	if item.Verified != nil {
		t.Fatalf("verified meta must be cleared on stale rejection: %+v", item.Verified)
	}
	var sawStale bool
	for _, ev := range item.Evidence {
		if ev.Event == "stale-verify" {
			sawStale = true
		}
	}
	if !sawStale {
		t.Fatal("stale-verify evidence not recorded")
	}
}

func TestListAndStatusSurfaceStaleVerification(t *testing.T) {
	opts, _ := testOptions(t, &fakeExec{def: execx.CmdResult{}})
	intakeOK(t, opts, "WI-001")
	if _, _, err := run(t, opts, "claim", "WI-001"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := run(t, opts, "verify", "WI-001"); err != nil {
		t.Fatal(err)
	}
	item, _ := opts.store().Load("WI-001")
	item.Checks = []string{"different"}
	if err := opts.store().Save(item); err != nil {
		t.Fatal(err)
	}

	stdout, _, err := run(t, opts, "list")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "verified(stale)") {
		t.Fatalf("list should flag stale verification: %q", stdout)
	}
	stdout, _, err = run(t, opts, "status")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "stale=1") {
		t.Fatalf("status should count stale verification: %q", stdout)
	}
}

func TestDeliverRejectsTamperedAcceptance(t *testing.T) {
	opts, _ := testOptions(t, &fakeExec{def: execx.CmdResult{}})
	intakeOK(t, opts, "WI-001")
	if _, _, err := run(t, opts, "claim", "WI-001"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := run(t, opts, "verify", "WI-001"); err != nil {
		t.Fatal(err)
	}
	item, _ := opts.store().Load("WI-001")
	item.Acceptance = append(item.Acceptance, "added later")
	if err := opts.store().Save(item); err != nil {
		t.Fatal(err)
	}
	if _, _, err := run(t, opts, "deliver", "WI-001"); err == nil {
		t.Fatal("deliver accepted tampered acceptance criteria")
	}
}

// TestDeliverAfterReverify shows the operator recovery path: after a stale
// rejection, a fresh verify re-binds the fingerprint and deliver succeeds.
func TestDeliverAfterReverify(t *testing.T) {
	opts, _ := testOptions(t, &fakeExec{def: execx.CmdResult{}})
	intakeOK(t, opts, "WI-001")
	if _, _, err := run(t, opts, "claim", "WI-001"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := run(t, opts, "verify", "WI-001"); err != nil {
		t.Fatal(err)
	}
	item, _ := opts.store().Load("WI-001")
	item.Checks = []string{"check-one"}
	if err := opts.store().Save(item); err != nil {
		t.Fatal(err)
	}
	if _, _, err := run(t, opts, "deliver", "WI-001"); err == nil {
		t.Fatal("expected stale rejection")
	}
	if _, _, err := run(t, opts, "verify", "WI-001"); err != nil {
		t.Fatalf("re-verify: %v", err)
	}
	if _, _, err := run(t, opts, "deliver", "WI-001"); err != nil {
		t.Fatalf("deliver after fresh verify: %v", err)
	}
}

// TestDeliverRejectsChangedSource reproduces the coordinator's fixture report:
// checks pass at verify, the tested source then changes (dirty, not committed),
// and deliver must refuse instead of shipping stale green evidence.
func TestDeliverRejectsChangedSource(t *testing.T) {
	exec := &fakeExec{def: execx.CmdResult{}}
	opts, _ := testOptions(t, exec)
	intakeOK(t, opts, "WI-024")
	if _, _, err := run(t, opts, "claim", "WI-024"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := run(t, opts, "verify", "WI-024"); err != nil {
		t.Fatal(err)
	}

	// Source drifts after verification: the same check commands now fail.
	exec.def = execx.CmdResult{ExitCode: 1, Stderr: "grep: no match"}
	_, _, err := run(t, opts, "deliver", "WI-024")
	if err == nil || !strings.Contains(err.Error(), "delivery re-check failed") {
		t.Fatalf("deliver err = %v, want delivery re-check rejection", err)
	}
	item, _ := opts.store().Load("WI-024")
	if item.State != StateInProgress || item.Verified != nil {
		t.Fatalf("state=%s verified=%+v, want in_progress with cleared verification", item.State, item.Verified)
	}
	var sawRecheck, sawStale bool
	for _, ev := range item.Evidence {
		if ev.Event == "deliver-check" {
			sawRecheck = true
			if ev.Log == "" {
				t.Fatalf("deliver-check evidence missing log path: %+v", ev)
			}
		}
		if ev.Event == "stale-verify" {
			sawStale = true
		}
	}
	if !sawRecheck || !sawStale {
		t.Fatalf("want deliver-check and stale-verify evidence, got %+v", item.Evidence)
	}

	// Recovery: source restored, fresh verify re-binds, deliver succeeds.
	exec.def = execx.CmdResult{}
	if _, _, err := run(t, opts, "verify", "WI-024"); err != nil {
		t.Fatalf("re-verify: %v", err)
	}
	stdout, _, err := run(t, opts, "deliver", "WI-024")
	if err != nil || !strings.Contains(stdout, "delivered WI-024") {
		t.Fatalf("deliver after fresh verify: out=%q err=%v", stdout, err)
	}
}

// TestDeliverRejectsMovedHead covers the committed variant of the fixture
// report: the recorded HEAD no longer matches, so delivery must refuse before
// re-running checks.
func TestDeliverRejectsMovedHead(t *testing.T) {
	exec := &fakeExec{def: execx.CmdResult{}}
	head := "abc1234"
	opts, _ := testOptionsWithHead(t, exec, func() string { return head })
	intakeOK(t, opts, "WI-024")
	if _, _, err := run(t, opts, "claim", "WI-024"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := run(t, opts, "verify", "WI-024"); err != nil {
		t.Fatal(err)
	}
	callsAtVerify := len(exec.calls)

	head = "def5678"
	_, _, err := run(t, opts, "deliver", "WI-024")
	if err == nil || !strings.Contains(err.Error(), "HEAD moved") {
		t.Fatalf("deliver err = %v, want HEAD-moved rejection", err)
	}
	if len(exec.calls) != callsAtVerify {
		t.Fatalf("moved-HEAD rejection must not run checks: calls %d -> %d", callsAtVerify, len(exec.calls))
	}
	item, _ := opts.store().Load("WI-024")
	if item.State != StateInProgress || item.Verified != nil {
		t.Fatalf("state=%s verified=%+v, want in_progress with cleared verification", item.State, item.Verified)
	}
}

// TestDeliverIdempotentWithoutRecheck ensures a repeated deliver on a delivered
// item is a pure no-op and does not re-run checks.
func TestDeliverIdempotentWithoutRecheck(t *testing.T) {
	exec := &fakeExec{def: execx.CmdResult{}}
	opts, _ := testOptions(t, exec)
	intakeOK(t, opts, "WI-001")
	for _, args := range [][]string{{"claim", "WI-001"}, {"verify", "WI-001"}, {"deliver", "WI-001"}} {
		if _, _, err := run(t, opts, args...); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	calls := len(exec.calls)
	if _, _, err := run(t, opts, "deliver", "WI-001"); err != nil {
		t.Fatal(err)
	}
	if len(exec.calls) != calls {
		t.Fatal("idempotent deliver re-ran checks")
	}
}

func TestVerifyFailurePropagates(t *testing.T) {
	exec := &fakeExec{
		def: execx.CmdResult{Stdout: "ok"},
		results: map[string]execx.CmdResult{
			"check-two": {ExitCode: 3, Stderr: "boom"},
		},
	}
	opts, _ := testOptions(t, exec)
	intakeOK(t, opts, "WI-010")
	if _, _, err := run(t, opts, "claim", "WI-010"); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := run(t, opts, "verify", "WI-010")
	if !errors.Is(err, errChecksFailed) {
		t.Fatalf("verify err = %v, want errChecksFailed", err)
	}
	if !strings.Contains(stderr, "check failed (exit 3): check-two") {
		t.Fatalf("stderr = %q", stderr)
	}
	item, _ := opts.store().Load("WI-010")
	if item.State != StateFailed {
		t.Fatalf("state = %s, want failed", item.State)
	}
	if item.LastError == "" {
		t.Fatal("last_error not recorded")
	}
	for _, ev := range item.Evidence {
		if ev.Event == "verify-pass" {
			t.Fatal("verify-pass recorded on failure")
		}
	}
	failCalls := 0
	for _, c := range exec.calls {
		if strings.Contains(c, "check-one") || strings.Contains(c, "check-two") {
			failCalls++
		}
	}
	if failCalls != 2 {
		t.Fatalf("exec calls for checks = %d, want 2 (stop at first failure)", failCalls)
	}

	// Fix the failing check, retry, and drive to delivered.
	delete(exec.results, "check-two")
	stdout, _, err := run(t, opts, "retry", "WI-010")
	if err != nil || !strings.Contains(stdout, "retried") {
		t.Fatalf("retry: out=%q err=%v", stdout, err)
	}
	item, _ = opts.store().Load("WI-010")
	if item.State != StateInProgress || item.Attempts != 2 {
		t.Fatalf("after retry: state=%s attempts=%d", item.State, item.Attempts)
	}
	if _, _, err := run(t, opts, "verify", "WI-010"); err != nil {
		t.Fatalf("re-verify: %v", err)
	}
	item, _ = opts.store().Load("WI-010")
	if item.LastError != "" {
		t.Fatalf("stale last_error after verify-pass: %q", item.LastError)
	}
	if _, _, err := run(t, opts, "deliver", "WI-010"); err != nil {
		t.Fatalf("deliver: %v", err)
	}
}

func TestVerifyStopsAtFirstFailure(t *testing.T) {
	exec := &fakeExec{
		results: map[string]execx.CmdResult{
			"check-one": {ExitCode: 1},
		},
		def: execx.CmdResult{},
	}
	opts, _ := testOptions(t, exec)
	intakeOK(t, opts, "WI-020")
	if _, _, err := run(t, opts, "claim", "WI-020"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := run(t, opts, "verify", "WI-020"); !errors.Is(err, errChecksFailed) {
		t.Fatalf("verify err = %v", err)
	}
	for _, c := range exec.calls {
		if strings.Contains(c, "check-two") {
			t.Fatal("second check ran after first failure")
		}
	}
	item, _ := opts.store().Load("WI-020")
	if len(item.Evidence) == 0 || item.Evidence[len(item.Evidence)-1].Event != "verify-fail" {
		t.Fatalf("last evidence = %+v", item.Evidence)
	}
}

func TestVerifyDryRun(t *testing.T) {
	exec := &fakeExec{def: execx.CmdResult{}}
	opts, dir := testOptions(t, exec)
	intakeOK(t, opts, "WI-030")
	if _, _, err := run(t, opts, "claim", "WI-030"); err != nil {
		t.Fatal(err)
	}
	stdout, _, err := run(t, opts, "verify", "WI-030", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "check-one") || !strings.Contains(stdout, "check-two") {
		t.Fatalf("dry-run output = %q", stdout)
	}
	if len(exec.calls) != 0 {
		t.Fatalf("dry-run executed commands: %v", exec.calls)
	}
	item, _ := opts.store().Load("WI-030")
	if item.State != StateInProgress || len(item.Evidence) != 1 {
		t.Fatalf("dry-run mutated item: %+v", item)
	}
	if _, err := os.Stat(filepath.Join(dir, ".factory", "run")); !os.IsNotExist(err) {
		t.Fatal("dry-run created run dir")
	}
}

func TestIllegalFlows(t *testing.T) {
	exec := &fakeExec{def: execx.CmdResult{}}
	opts, _ := testOptions(t, exec)
	intakeOK(t, opts, "WI-040")

	if _, _, err := run(t, opts, "verify", "WI-040"); err == nil {
		t.Fatal("verify on queued should fail")
	}
	if _, _, err := run(t, opts, "deliver", "WI-040"); err == nil {
		t.Fatal("deliver on queued should fail")
	}
	if _, _, err := run(t, opts, "claim", "WI-040"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := run(t, opts, "deliver", "WI-040"); err == nil {
		t.Fatal("deliver on in_progress should fail")
	}
	if _, _, err := run(t, opts, "unblock", "WI-040"); err == nil {
		t.Fatal("unblock on in_progress should fail")
	}
	if _, _, err := run(t, opts, "block", "WI-040"); err == nil {
		t.Fatal("block without reason should fail")
	}
	if _, _, err := run(t, opts, "evidence", "WI-040"); err == nil {
		t.Fatal("evidence without note should fail")
	}
}

func TestBlockUnblockCycle(t *testing.T) {
	exec := &fakeExec{def: execx.CmdResult{}}
	opts, _ := testOptions(t, exec)
	intakeOK(t, opts, "WI-050")

	stdout, _, err := run(t, opts, "block", "WI-050", "--reason", "waiting on upstream")
	if err != nil || !strings.Contains(stdout, "blocked WI-050") {
		t.Fatalf("block: out=%q err=%v", stdout, err)
	}
	item, _ := opts.store().Load("WI-050")
	if item.State != StateBlocked || item.LastError != "waiting on upstream" {
		t.Fatalf("blocked item = %+v", item)
	}
	stdout, _, err = run(t, opts, "block", "WI-050", "--reason", "again")
	if err != nil || !strings.Contains(stdout, "already blocked") {
		t.Fatalf("idempotent block: out=%q err=%v", stdout, err)
	}
	stdout, _, err = run(t, opts, "unblock", "WI-050")
	if err != nil || !strings.Contains(stdout, "queued") {
		t.Fatalf("unblock: out=%q err=%v", stdout, err)
	}
	item, _ = opts.store().Load("WI-050")
	if item.State != StateQueued || item.LastError != "" {
		t.Fatalf("unblocked item = %+v", item)
	}
}

func TestIntakeValidation(t *testing.T) {
	opts, _ := testOptions(t, &fakeExec{})

	cases := []struct {
		name string
		args []string
	}{
		{"no checks", []string{"intake", "--id", "WI-1", "--title", "t", "--acceptance", "a"}},
		{"no acceptance", []string{"intake", "--id", "WI-1", "--title", "t", "--check", "c"}},
		{"bad id", []string{"intake", "--id", "wi-1", "--title", "t", "--acceptance", "a", "--check", "c"}},
		{"bad kind", []string{"intake", "--id", "WI-1", "--title", "t", "--kind", "epic", "--acceptance", "a", "--check", "c"}},
	}
	for _, tc := range cases {
		if _, _, err := run(t, opts, tc.args...); err == nil {
			t.Errorf("%s: expected error", tc.name)
		}
	}

	intakeOK(t, opts, "WI-001")
	if _, _, err := run(t, opts, "intake", "--id", "WI-001", "--title", "dup",
		"--acceptance", "a", "--check", "c"); err == nil {
		t.Fatal("duplicate intake should fail")
	}
}

func TestClaimOnFailedRequiresRetry(t *testing.T) {
	exec := &fakeExec{
		results: map[string]execx.CmdResult{"check-one": {ExitCode: 1}},
	}
	opts, _ := testOptions(t, exec)
	intakeOK(t, opts, "WI-060")
	if _, _, err := run(t, opts, "claim", "WI-060"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := run(t, opts, "verify", "WI-060"); err == nil {
		t.Fatal("verify should fail")
	}
	_, _, err := run(t, opts, "claim", "WI-060")
	if err == nil || !strings.Contains(err.Error(), "retry") {
		t.Fatalf("claim on failed: %v", err)
	}
}

func TestRetryOnlyAppliesToFailed(t *testing.T) {
	opts, _ := testOptions(t, &fakeExec{})
	intakeOK(t, opts, "WI-070")
	if _, _, err := run(t, opts, "retry", "WI-070"); err == nil {
		t.Fatal("retry on queued should fail")
	}
}

func TestListJSONAndStatusJSON(t *testing.T) {
	opts, _ := testOptions(t, &fakeExec{})
	intakeOK(t, opts, "WI-080")

	stdout, _, err := run(t, opts, "list", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var items []WorkItem
	if err := json.Unmarshal([]byte(stdout), &items); err != nil {
		t.Fatalf("list --json: %v", err)
	}
	if len(items) != 1 || items[0].ID != "WI-080" {
		t.Fatalf("items = %+v", items)
	}

	stdout, _, err = run(t, opts, "list", "--state", "delivered")
	if err != nil || strings.Contains(stdout, "WI-080") {
		t.Fatalf("filtered list = %q", stdout)
	}
	stdout, _, err = run(t, opts, "status", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var status struct {
		Total  int           `json:"total"`
		Counts map[State]int `json:"counts"`
	}
	if err := json.Unmarshal([]byte(stdout), &status); err != nil {
		t.Fatalf("status --json: %v", err)
	}
	if status.Total != 1 || status.Counts[StateQueued] != 1 {
		t.Fatalf("status = %+v", status)
	}
}

func TestNextEmptyAndNextJSON(t *testing.T) {
	opts, _ := testOptions(t, &fakeExec{})
	stdout, _, err := run(t, opts, "next")
	if err != nil || !strings.Contains(stdout, "no queued items") {
		t.Fatalf("next empty = %q err=%v", stdout, err)
	}
	intakeOK(t, opts, "WI-090")
	stdout, _, err = run(t, opts, "next", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var item WorkItem
	if err := json.Unmarshal([]byte(stdout), &item); err != nil || item.ID != "WI-090" {
		t.Fatalf("next --json: %v %+v", err, item)
	}
}

func TestEvidenceNote(t *testing.T) {
	opts, _ := testOptions(t, &fakeExec{})
	intakeOK(t, opts, "WI-100")
	if _, _, err := run(t, opts, "evidence", "WI-100", "--note", "investigated"); err != nil {
		t.Fatal(err)
	}
	item, _ := opts.store().Load("WI-100")
	if len(item.Evidence) != 1 || item.Evidence[0].Event != "note" || item.Evidence[0].Head != "abc1234" {
		t.Fatalf("evidence = %+v", item.Evidence)
	}
}

func TestLoadMissingItemErrors(t *testing.T) {
	opts, _ := testOptions(t, &fakeExec{})
	for _, args := range [][]string{
		{"claim", "WI-999"}, {"verify", "WI-999"}, {"deliver", "WI-999"},
		{"block", "WI-999", "--reason", "x"}, {"unblock", "WI-999"}, {"retry", "WI-999"},
		{"evidence", "WI-999", "--note", "x"},
	} {
		if _, _, err := run(t, opts, args...); err == nil {
			t.Errorf("%v: expected error for missing item", args)
		}
	}
}

func TestExecuteBoundary(t *testing.T) {
	var out, errOut bytes.Buffer
	err := Execute(context.Background(), []string{"bogus-command"}, &out, &errOut)
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
	err = Execute(context.Background(), []string{"--help"}, &out, &errOut)
	if err != nil {
		t.Fatalf("--help should succeed: %v", err)
	}
	if !strings.Contains(out.String(), "queued -> in_progress") {
		t.Fatalf("help output missing pipeline description: %q", out.String())
	}
}

func TestSlugify(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"Add snapshot-diff schema!": "add-snapshot-diff-schema",
		"  spaces  EVERYWHERE  ":    "spaces-everywhere",
		"already-good-123":          "already-good-123",
		"!!!":                       "",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestListAndStatusSurfaceMovedHead pins WI-026: verification recorded at an
// older commit renders stale before any delivery attempt.
func TestListAndStatusSurfaceMovedHead(t *testing.T) {
	head := "abc1234"
	opts, _ := testOptionsWithHead(t, &fakeExec{def: execx.CmdResult{}}, func() string { return head })
	intakeOK(t, opts, "WI-001")
	for _, args := range [][]string{{"claim", "WI-001"}, {"verify", "WI-001"}} {
		if _, _, err := run(t, opts, args...); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}

	stdout, _, err := run(t, opts, "list")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "verified ") || strings.Contains(stdout, "stale") {
		t.Fatalf("fresh verification should not be stale: %q", stdout)
	}

	head = "def5678"
	stdout, _, err = run(t, opts, "list")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "verified(stale)") {
		t.Fatalf("moved HEAD should render verified(stale): %q", stdout)
	}
	stdout, _, err = run(t, opts, "status")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "stale=1") {
		t.Fatalf("status should count moved-head staleness: %q", stdout)
	}
}

// TestVerifyInfraErrorLandsInFailed pins WI-028: when the check run itself
// cannot complete (here the run dir cannot be created), the item must not be
// left parked in the transient verifying state.
func TestVerifyInfraErrorLandsInFailed(t *testing.T) {
	opts, dir := testOptions(t, &fakeExec{def: execx.CmdResult{}})
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	opts.exec.RunDir = blocker

	intakeOK(t, opts, "WI-028")
	if _, _, err := run(t, opts, "claim", "WI-028"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := run(t, opts, "verify", "WI-028"); err == nil {
		t.Fatal("verify should propagate the infrastructure error")
	}
	item, _ := opts.store().Load("WI-028")
	if item.State != StateFailed {
		t.Fatalf("state = %s, want failed after infra error", item.State)
	}
	if !strings.Contains(item.LastError, "verification could not complete") {
		t.Fatalf("last_error = %q", item.LastError)
	}
	var sawFail bool
	for _, ev := range item.Evidence {
		if ev.Event == "verify-fail" {
			sawFail = true
		}
	}
	if !sawFail {
		t.Fatal("verify-fail evidence not recorded")
	}
	// Recovery path stays the standard retry -> verify.
	if _, _, err := run(t, opts, "retry", "WI-028"); err != nil {
		t.Fatalf("retry: %v", err)
	}
}

func makeBacklogReadonly(t *testing.T, dir string) {
	t.Helper()
	backlog := filepath.Join(dir, ".factory", "backlog")
	if err := os.MkdirAll(backlog, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(backlog, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(backlog, 0o755) })
}

// TestRunnerFillInBranches covers options.runner defaulting empty fields on an
// injected Runner.
func TestRunnerFillInBranches(t *testing.T) {
	dir := t.TempDir()
	o := &options{
		root: dir,
		now:  func() time.Time { return time.Unix(0, 0) },
		exec: Runner{Exec: &fakeExec{}},
	}
	r := o.runner()
	if r.RunDir == "" || r.Root != dir || r.Now == nil {
		t.Fatalf("runner fields not filled: %+v", r)
	}
}

func TestExecuteNilContext(t *testing.T) {
	dir := t.TempDir()
	var out, errOut bytes.Buffer
	// nil ctx falls back to context.Background(); status on an empty root
	// still runs and reports zero items.
	var nilCtx context.Context
	if err := Execute(nilCtx, []string{"--root", dir, "status"}, &out, &errOut); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "items=0") {
		t.Fatalf("status output = %q", out.String())
	}
}

func TestClaimSaveErrorPropagates(t *testing.T) {
	opts, dir := testOptions(t, &fakeExec{def: execx.CmdResult{}})
	intakeOK(t, opts, "WI-001")
	makeBacklogReadonly(t, dir)
	if _, _, err := run(t, opts, "claim", "WI-001"); err == nil {
		t.Fatal("claim should fail when the item cannot be saved")
	}
}

func TestVerifySaveErrorPropagates(t *testing.T) {
	opts, dir := testOptions(t, &fakeExec{def: execx.CmdResult{}})
	intakeOK(t, opts, "WI-001")
	if _, _, err := run(t, opts, "claim", "WI-001"); err != nil {
		t.Fatal(err)
	}
	makeBacklogReadonly(t, dir)
	if _, _, err := run(t, opts, "verify", "WI-001"); err == nil {
		t.Fatal("verify should fail when the item cannot be saved")
	}
}

// TestVerifyInfraAndSaveError covers the save-failure branch inside the
// infra-error path: the run cannot start AND the state write fails.
func TestVerifyInfraAndSaveError(t *testing.T) {
	opts, dir := testOptions(t, &fakeExec{def: execx.CmdResult{}})
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	opts.exec.RunDir = blocker
	intakeOK(t, opts, "WI-028")
	if _, _, err := run(t, opts, "claim", "WI-028"); err != nil {
		t.Fatal(err)
	}
	makeBacklogReadonly(t, dir)
	if _, _, err := run(t, opts, "verify", "WI-028"); err == nil {
		t.Fatal("verify should fail")
	}
}

// TestDeliverRunChecksInfraError covers delivery when the re-check run itself
// cannot start; the item must stay verified for a later retry.
func TestDeliverRunChecksInfraError(t *testing.T) {
	opts, dir := testOptions(t, &fakeExec{def: execx.CmdResult{}})
	intakeOK(t, opts, "WI-001")
	for _, args := range [][]string{{"claim", "WI-001"}, {"verify", "WI-001"}} {
		if _, _, err := run(t, opts, args...); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	opts.exec.RunDir = blocker
	if _, _, err := run(t, opts, "deliver", "WI-001"); err == nil {
		t.Fatal("deliver should propagate the re-check infrastructure error")
	}
	item, _ := opts.store().Load("WI-001")
	if item.State != StateVerified {
		t.Fatalf("state = %s, want still verified after infra error", item.State)
	}
}

func TestDeliverRejectsStaleSaveError(t *testing.T) {
	opts, dir := testOptions(t, &fakeExec{def: execx.CmdResult{}})
	intakeOK(t, opts, "WI-001")
	for _, args := range [][]string{{"claim", "WI-001"}, {"verify", "WI-001"}} {
		if _, _, err := run(t, opts, args...); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	item, _ := opts.store().Load("WI-001")
	item.Checks = []string{"edited-after-verify"}
	if err := opts.store().Save(item); err != nil {
		t.Fatal(err)
	}
	makeBacklogReadonly(t, dir)
	if _, _, err := run(t, opts, "deliver", "WI-001"); err == nil {
		t.Fatal("deliver should fail when the stale rollback cannot be saved")
	}
}

func TestClaimDryRunDoesNotMutate(t *testing.T) {
	opts, _ := testOptions(t, &fakeExec{def: execx.CmdResult{}})
	intakeOK(t, opts, "WI-001")
	stdout, _, err := run(t, opts, "claim", "WI-001", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "dry-run") {
		t.Fatalf("claim dry-run output = %q", stdout)
	}
	item, _ := opts.store().Load("WI-001")
	if item.State != StateQueued || item.Attempts != 0 {
		t.Fatalf("dry-run claim mutated item: state=%s attempts=%d", item.State, item.Attempts)
	}
}

func TestListFailsOnUnreadableBacklog(t *testing.T) {
	opts, dir := testOptions(t, &fakeExec{def: execx.CmdResult{}})
	blocker := filepath.Join(dir, ".factory")
	if err := os.MkdirAll(blocker, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(blocker, "backlog"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := run(t, opts, "list"); err == nil {
		t.Fatal("list should fail when backlog dir is unreadable")
	}
}

// TestVerifiedMetadataClearedWhenLeavingState pins WI-033: verified metadata is
// provenance of the verified/delivered states only; any other move drops it.
func TestVerifiedMetadataClearedWhenLeavingState(t *testing.T) {
	exec := &fakeExec{def: execx.CmdResult{}}
	opts, _ := testOptions(t, exec)
	intakeOK(t, opts, "WI-033")
	for _, args := range [][]string{{"claim", "WI-033"}, {"verify", "WI-033"}} {
		if _, _, err := run(t, opts, args...); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	if item, _ := opts.store().Load("WI-033"); item.Verified == nil {
		t.Fatal("expected verified metadata after pass")
	}

	// Blocking a verified item clears the metadata.
	if _, _, err := run(t, opts, "block", "WI-033", "--reason", "park"); err != nil {
		t.Fatal(err)
	}
	if item, _ := opts.store().Load("WI-033"); item.Verified != nil {
		t.Fatalf("verified metadata left on state=%s", item.State)
	}
}

// TestReverifyFailureDropsVerifiedMetadata covers verify -> re-verify -> fail:
// the old verified block must not survive on the failed item.
func TestReverifyFailureDropsVerifiedMetadata(t *testing.T) {
	exec := &fakeExec{def: execx.CmdResult{}}
	opts, _ := testOptions(t, exec)
	intakeOK(t, opts, "WI-033")
	for _, args := range [][]string{{"claim", "WI-033"}, {"verify", "WI-033"}} {
		if _, _, err := run(t, opts, args...); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	exec.def = execx.CmdResult{ExitCode: 1, Stderr: "boom"}
	if _, _, err := run(t, opts, "verify", "WI-033"); err == nil {
		t.Fatal("re-verify should fail")
	}
	item, _ := opts.store().Load("WI-033")
	if item.State != StateFailed || item.Verified != nil {
		t.Fatalf("state=%s verified=%+v, want failed without verified metadata", item.State, item.Verified)
	}
}

// TestBlockDeliveredItemFails covers the transition-error path: delivered is
// terminal, so blocking it must fail without mutating the item.
func TestBlockDeliveredItemFails(t *testing.T) {
	opts, _ := testOptions(t, &fakeExec{def: execx.CmdResult{}})
	intakeOK(t, opts, "WI-036")
	for _, args := range [][]string{{"claim", "WI-036"}, {"verify", "WI-036"}, {"deliver", "WI-036"}} {
		if _, _, err := run(t, opts, args...); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	if _, _, err := run(t, opts, "block", "WI-036", "--reason", "late"); err == nil {
		t.Fatal("block on delivered item should fail")
	}
	item, _ := opts.store().Load("WI-036")
	if item.State != StateDelivered {
		t.Fatalf("state = %s, want delivered", item.State)
	}
}

func TestBlockDryRunAndSaveError(t *testing.T) {
	opts, dir := testOptions(t, &fakeExec{def: execx.CmdResult{}})
	intakeOK(t, opts, "WI-036")

	stdout, _, err := run(t, opts, "block", "WI-036", "--reason", "preview", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "dry-run: would block") {
		t.Fatalf("dry-run output = %q", stdout)
	}
	item, _ := opts.store().Load("WI-036")
	if item.State != StateQueued {
		t.Fatalf("dry-run mutated state to %s", item.State)
	}

	makeBacklogReadonly(t, dir)
	if _, _, err := run(t, opts, "block", "WI-036", "--reason", "x"); err == nil {
		t.Fatal("block should fail when save fails")
	}
}

func TestUnblockAndEvidenceSaveErrors(t *testing.T) {
	opts, dir := testOptions(t, &fakeExec{def: execx.CmdResult{}})
	intakeOK(t, opts, "WI-036")
	if _, _, err := run(t, opts, "block", "WI-036", "--reason", "park"); err != nil {
		t.Fatal(err)
	}
	makeBacklogReadonly(t, dir)
	if _, _, err := run(t, opts, "unblock", "WI-036"); err == nil {
		t.Fatal("unblock should fail when save fails")
	}
	if _, _, err := run(t, opts, "evidence", "WI-036", "--note", "x"); err == nil {
		t.Fatal("evidence should fail when save fails")
	}
}

func TestStatusListErrorPropagates(t *testing.T) {
	opts, dir := testOptions(t, &fakeExec{def: execx.CmdResult{}})
	blocker := filepath.Join(dir, ".factory", "backlog")
	if err := os.MkdirAll(filepath.Dir(blocker), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := run(t, opts, "status"); err == nil {
		t.Fatal("status should fail when the backlog dir is not listable")
	}
}

func TestValidateCommandPropagatesFailure(t *testing.T) {
	opts, _ := testOptions(t, &fakeExec{def: execx.CmdResult{}})
	// No schema file under the temp root -> ValidateBacklog fails.
	if _, _, err := run(t, opts, "validate"); err == nil {
		t.Fatal("validate should fail without a schema")
	}
}

// TestClaimVerifiedItemRefused pins WI-040: verified -> in_progress exists for
// stale-verify rollback only; a plain claim must not silently demote.
func TestClaimVerifiedItemRefused(t *testing.T) {
	opts, _ := testOptions(t, &fakeExec{def: execx.CmdResult{}})
	intakeOK(t, opts, "WI-040")
	for _, args := range [][]string{{"claim", "WI-040"}, {"verify", "WI-040"}} {
		if _, _, err := run(t, opts, args...); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	_, _, err := run(t, opts, "claim", "WI-040")
	if err == nil || !strings.Contains(err.Error(), "is verified") {
		t.Fatalf("claim verified err = %v", err)
	}
	item, _ := opts.store().Load("WI-040")
	if item.State != StateVerified || item.Verified == nil {
		t.Fatalf("state=%s verified=%+v after refused claim", item.State, item.Verified)
	}
	// And the sibling path stays correct: queued claims still work.
	intakeOK(t, opts, "WI-041")
	if _, _, err := run(t, opts, "claim", "WI-041"); err != nil {
		t.Fatalf("claim queued: %v", err)
	}
}
