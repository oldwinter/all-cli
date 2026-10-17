package tools

import (
	"context"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/oldwinter/all-cli/internal/execx"
)

// dupRecordingRunner records every command issued during a measurement window.
type dupRecordingRunner struct {
	mu    sync.Mutex
	calls []string
}

func (r *dupRecordingRunner) Run(_ context.Context, name string, args ...string) execx.CmdResult {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, name+" "+strings.Join(args, " "))
	return execx.CmdResult{}
}

func (r *dupRecordingRunner) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = nil
}

func (r *dupRecordingRunner) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.calls)
}

// duplicateCount reports how many recorded calls repeat an earlier identical one.
func (r *dupRecordingRunner) duplicateCount() (total, duplicates int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	seen := map[string]bool{}
	for _, call := range r.calls {
		if seen[call] {
			duplicates++
			continue
		}
		seen[call] = true
	}
	return len(r.calls), duplicates
}

func (r *dupRecordingRunner) sortedCalls() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]string, len(r.calls))
	copy(out, r.calls)
	sort.Strings(out)
	return out
}

// adapterToolIDs lists the registry tools whose ConfigCheck and Current both
// shell out. It is the exact set the P1-1 finding named.
var adapterToolIDs = []string{
	"aws", "aliyun", "wrangler", "vercel", "railway", "netlify",
	"kubectl", "docker", "gh", "glab", "kargo", "argocd", "opencli",
}

// runBothCallbacks mirrors what Evaluate does after LookPath succeeds: run
// ConfigCheck, then Current, through the supplied runner.
func runBothCallbacks(ctx context.Context, def ToolDefinition, runner execx.Runner) {
	def.ConfigCheck(ctx, runner, true)
	def.Current(ctx, runner, true)
}

// TestEvaluateSubprocessDuplicateMeasurement is the measurement harness for the
// P1-1 fix. For every adapter-backed tool it runs the Configured+Current pair
// twice: once with the bare runner (the pre-fix behaviour) and once through the
// memoizing runner Evaluate installs (the fixed behaviour). The bare run shows
// the duplicates; the memoized run must show none.
func TestEvaluateSubprocessDuplicateMeasurement(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	runner := &dupRecordingRunner{}

	beforeTotal, beforeDuplicates := 0, 0
	afterTotal, afterDuplicates := 0, 0

	for _, id := range adapterToolIDs {
		def, ok := FindByID(id)
		if !ok {
			t.Fatalf("%s missing from registry", id)
		}

		// BEFORE: no memo, which is what Evaluate did before the fix.
		runner.reset()
		runBothCallbacks(ctx, def, runner)
		total, duplicates := runner.duplicateCount()
		beforeTotal += total
		beforeDuplicates += duplicates

		// AFTER: the memoizing runner Evaluate now installs.
		runner.reset()
		runBothCallbacks(ctx, def, memoRunner{scope: newEvalScope(), inner: runner})
		fixedTotal, fixedDuplicates := runner.duplicateCount()
		afterTotal += fixedTotal
		afterDuplicates += fixedDuplicates

		if fixedDuplicates != 0 {
			t.Errorf("%s still issued %d duplicate command(s) out of %d: %#v",
				id, fixedDuplicates, fixedTotal, runner.sortedCalls())
		}

		t.Logf("%-9s before: %d calls (%d dup)  after: %d calls (%d dup)",
			id, total, duplicates, fixedTotal, fixedDuplicates)
	}

	if beforeDuplicates == 0 {
		t.Fatal("expected the unmemoized path to duplicate commands; harness is not measuring the bug")
	}

	t.Logf("BEFORE (no memo): %d subprocess calls, %d duplicates (%d unique)",
		beforeTotal, beforeDuplicates, beforeTotal-beforeDuplicates)
	t.Logf("AFTER  (memo):    %d subprocess calls, %d duplicates (%d unique)",
		afterTotal, afterDuplicates, afterTotal-afterDuplicates)
	t.Logf("reduction: %d -> %d calls (%.0f%% fewer)",
		beforeTotal, afterTotal, 100*float64(beforeTotal-afterTotal)/float64(beforeTotal))
}

// TestEvalScopeRunReplaysSameCommand pins the memo semantics directly.
func TestEvalScopeRunReplaysSameCommand(t *testing.T) {
	t.Parallel()

	runner := &dupRecordingRunner{}
	scope := newEvalScope()
	ctx := context.Background()

	first := scope.run(ctx, runner, "vercel", "whoami", "--format", "json")
	second := scope.run(ctx, runner, "vercel", "whoami", "--format", "json")
	if first != second {
		t.Fatalf("replayed result = %#v, want %#v", second, first)
	}
	if got := runner.count(); got != 1 {
		t.Fatalf("runner calls = %d, want 1", got)
	}

	// A different argument vector must not be served from the first entry.
	scope.run(ctx, runner, "vercel", "whoami")
	if got := runner.count(); got != 2 {
		t.Fatalf("runner calls = %d after distinct args, want 2", got)
	}
}

// TestEvalScopeIsPerEvaluation proves a second Evaluate never reuses the first
// evaluation's recorded commands.
func TestEvalScopeIsPerEvaluation(t *testing.T) {
	t.Parallel()

	runner := &dupRecordingRunner{}
	scope := newEvalScope()

	scope.run(context.Background(), runner, "gh", "auth", "status")
	if got := runner.count(); got != 1 {
		t.Fatalf("runner calls = %d, want 1", got)
	}

	fresh := newEvalScope()
	fresh.run(context.Background(), runner, "gh", "auth", "status")
	if got := runner.count(); got != 2 {
		t.Fatalf("runner calls = %d, want 2 (a fresh scope must not replay)", got)
	}
}

// TestEvalScopeConcurrentCallersShareOneExecution exercises the shared map and
// the per-cell sync.Once from several goroutines at once.
//
// This is the shape status.go uses: every registry tool is evaluated in its own
// goroutine. The race detector is the ideal check for this, but it needs cgo and
// a C compiler, which not every CI host has, so the invariant is also asserted
// directly: N concurrent callers must produce exactly one execution and all of
// them must observe the identical result. A broken sync.Once or an unsynchronized
// map access shows up here as either a wrong call count or a panic.
func TestEvalScopeConcurrentCallersShareOneExecution(t *testing.T) {
	t.Parallel()

	const goroutines = 16

	runner := &dupRecordingRunner{}
	scope := newEvalScope()
	ctx := context.Background()

	var wg sync.WaitGroup
	results := make([]execx.CmdResult, goroutines)
	wg.Add(goroutines)
	for i := range goroutines {
		go func(i int) {
			defer wg.Done()
			results[i] = scope.run(ctx, runner, "kubectl", "config", "get-contexts", "-o", "name")
		}(i)
	}
	wg.Wait()

	if got := runner.count(); got != 1 {
		t.Fatalf("runner calls = %d, want exactly 1 for %d concurrent callers", got, goroutines)
	}
	for i, res := range results {
		if res != results[0] {
			t.Fatalf("caller %d observed %#v, want %#v", i, res, results[0])
		}
	}
}

// TestEvaluateConcurrentToolEvaluationsDoNotShareCommands mirrors how
// evaluateStatusRegistry fans tools out across goroutines and asserts that each
// evaluation keeps its own memo. Two definitions that run the same command must
// still execute it once each, never once total.
func TestEvaluateConcurrentToolEvaluationsDoNotShareCommands(t *testing.T) {
	t.Parallel()

	const evaluations = 8

	runner := &dupRecordingRunner{}
	var wg sync.WaitGroup
	wg.Add(evaluations)
	for range evaluations {
		go func() {
			defer wg.Done()
			scope := newEvalScope()
			scope.run(context.Background(), runner, "vercel", "whoami")
		}()
	}
	wg.Wait()

	// One shared scope per goroutine: every goroutine owns its own, so the same
	// command runs once per evaluation.
	if got := runner.count(); got != evaluations {
		t.Fatalf("runner calls = %d, want %d (one per independent evaluation)", got, evaluations)
	}
}
