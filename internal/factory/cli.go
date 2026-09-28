package factory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/spf13/cobra"
)

type options struct {
	root string
	json bool
	dry  bool
	now  func() time.Time
	head func() string
	exec Runner
}

// Execute runs the factory CLI. It mirrors internal/cli.Execute's boundary:
// args without the program name, explicit writers, error return for exit code.
func Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if ctx == nil {
		ctx = context.Background()
	}
	opts := &options{now: time.Now}
	root := newRootCommand(opts)
	root.SetContext(ctx)
	root.SetArgs(args)
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SilenceUsage = true
	return root.Execute()
}

func newRootCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "factory",
		Short: "Repository work-item factory: intake, claim, verify, deliver",
		Long: `factory moves JSON work items in .factory/backlog through an ordered
pipeline: queued -> in_progress -> verifying -> verified -> delivered.
Verification checks recorded on each item run for real; a failed check moves
the item to failed and returns a non-zero exit code. Run logs land in
.factory/run/ (ignored by git); the backlog files themselves are reviewed
in git like any other source.`,
	}
	cmd.PersistentFlags().StringVar(&opts.root, "root", opts.root, "Repository root containing .factory/")
	cmd.PersistentFlags().BoolVar(&opts.json, "json", opts.json, "Emit machine-readable JSON where supported")
	cmd.AddCommand(
		newListCommand(opts),
		newNextCommand(opts),
		newIntakeCommand(opts),
		newClaimCommand(opts),
		newRetryCommand(opts),
		newVerifyCommand(opts),
		newDeliverCommand(opts),
		newBlockCommand(opts),
		newUnblockCommand(opts),
		newEvidenceCommand(opts),
		newStatusCommand(opts),
		newValidateCommand(opts),
	)
	return cmd
}

func (o *options) store() Store {
	return NewStore(filepath.Join(o.root, ".factory", "backlog"))
}

func (o *options) runner() Runner {
	r := o.exec
	if r.Exec == nil {
		r = NewRunner(o.root)
	}
	if r.RunDir == "" {
		r.RunDir = filepath.Join(o.root, ".factory", "run")
	}
	if r.Root == "" {
		r.Root = o.root
	}
	if r.Now == nil {
		r.Now = o.now
	}
	if r.Head == nil {
		r.Head = o.headFunc()
	}
	return r
}

func (o *options) headFunc() func() string {
	if o.head != nil {
		return o.head
	}
	return gitHead(o.root, execx.TimeoutRunner{
		Runner:  execx.DefaultRunner{},
		Timeout: 15 * time.Second,
	})
}

func (o *options) load(id string) (*WorkItem, error) {
	return o.store().Load(id)
}

func (o *options) save(item *WorkItem) error {
	return o.store().Save(item)
}

func printJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func newListCommand(opts *options) *cobra.Command {
	var state string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List backlog items by state",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			items, err := opts.store().List()
			if err != nil {
				return err
			}
			var filtered []*WorkItem
			for _, it := range items {
				if state == "" || string(it.State) == state {
					filtered = append(filtered, it)
				}
			}
			if opts.json {
				return printJSON(cmd.OutOrStdout(), filtered)
			}
			for _, it := range filtered {
				fmt.Fprintf(cmd.OutOrStdout(), "%-8s %-12s ord=%-3d attempts=%d %s\n",
					it.ID, it.State, it.Order, it.Attempts, it.Title)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&state, "state", "", "Filter by state (queued|in_progress|verifying|verified|delivered|failed|blocked)")
	return cmd
}

func newNextCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "next",
		Short: "Print the next queued work item",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			item, err := opts.store().Next()
			if err != nil {
				return err
			}
			if item == nil {
				fmt.Fprintln(cmd.OutOrStdout(), "no queued items")
				return nil
			}
			if opts.json {
				return printJSON(cmd.OutOrStdout(), item)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", item.ID, item.Title)
			for _, a := range item.Acceptance {
				fmt.Fprintf(cmd.OutOrStdout(), "  accept: %s\n", a)
			}
			for _, c := range item.Checks {
				fmt.Fprintf(cmd.OutOrStdout(), "  check:  %s\n", c)
			}
			return nil
		},
	}
}

func slugify(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else if b.Len() > 0 && b.String()[b.Len()-1] != '-' {
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func newIntakeCommand(opts *options) *cobra.Command {
	var id, title, kind, branch string
	var order, issue int
	var acceptance, checks []string
	var avoidPRs []int
	cmd := &cobra.Command{
		Use:   "intake --id WI-NNN --title T --acceptance A --check C",
		Short: "Create a queued work item",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if branch == "" {
				branch = "factory/" + strings.ToLower(id) + "-" + slugify(title)
			}
			item := &WorkItem{
				SchemaVersion: ItemSchemaVersion,
				ID:            id,
				Title:         title,
				Kind:          kind,
				State:         StateQueued,
				Order:         order,
				Source:        Source{Issue: issue, AvoidPRs: avoidPRs},
				Acceptance:    acceptance,
				Checks:        checks,
				Branch:        branch,
			}
			now := opts.now()
			item.CreatedAt = now.UTC().Format(time.RFC3339)
			item.UpdatedAt = item.CreatedAt
			if err := item.Validate(); err != nil {
				return err
			}
			if opts.store().Exists(id) {
				return fmt.Errorf("work item %s already exists", id)
			}
			if opts.dry {
				fmt.Fprintf(cmd.OutOrStdout(), "dry-run: would create %s\n", item.ID)
				return nil
			}
			if err := opts.save(item); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "queued %s (%s)\n", item.ID, item.Title)
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "Work item ID (e.g. WI-001)")
	cmd.Flags().StringVar(&title, "title", "", "Short work item title")
	cmd.Flags().StringVar(&kind, "kind", "improvement", "improvement|bug|test|docs")
	cmd.Flags().IntVar(&order, "order", 100, "Queue order (lower runs first)")
	cmd.Flags().IntVar(&issue, "issue", 0, "Linked GitHub issue number, if any")
	cmd.Flags().IntSliceVar(&avoidPRs, "avoid-pr", nil, "Open PR numbers this item must not duplicate")
	cmd.Flags().StringVar(&branch, "branch", "", "Delivery branch name (default factory/<id>-<slug>)")
	cmd.Flags().StringArrayVar(&acceptance, "acceptance", nil, "Acceptance criterion (repeatable)")
	cmd.Flags().StringArrayVar(&checks, "check", nil, "Verification command run from repo root (repeatable)")
	cmd.Flags().BoolVar(&opts.dry, "dry-run", false, "Show what would happen without writing")
	return cmd
}

func claimItem(opts *options, id string, w io.Writer, retry bool) error {
	item, err := opts.load(id)
	if err != nil {
		return err
	}
	verb := "claimed"
	if retry {
		if item.State != StateFailed && item.State != StateInProgress {
			return fmt.Errorf("item %s: retry applies to failed items (state=%s)", id, item.State)
		}
	} else if item.State == StateFailed {
		return fmt.Errorf("item %s is failed; use 'factory retry %s'", id, id)
	}
	if item.State == StateInProgress {
		fmt.Fprintf(w, "%s already in_progress (attempts=%d)\n", id, item.Attempts)
		return nil
	}
	if err := item.Transition(StateInProgress, opts.now()); err != nil {
		return err
	}
	item.Attempts++
	if retry {
		verb = "retried"
	}
	item.Record(Evidence{
		Event: verb,
		Note:  fmt.Sprintf("attempt %d", item.Attempts),
		Head:  opts.headFunc()(),
	}, opts.now())
	if opts.dry {
		fmt.Fprintf(w, "dry-run: would move %s to in_progress\n", id)
		return nil
	}
	if err := opts.save(item); err != nil {
		return err
	}
	fmt.Fprintf(w, "%s %s -> in_progress (attempt %d, branch %s)\n", verb, id, item.Attempts, item.Branch)
	return nil
}

func newClaimCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claim <id>",
		Short: "Move a queued item to in_progress",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return claimItem(opts, args[0], cmd.OutOrStdout(), false)
		},
	}
	cmd.Flags().BoolVar(&opts.dry, "dry-run", false, "Show transition without writing")
	return cmd
}

func newRetryCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "retry <id>",
		Short: "Move a failed item back to in_progress",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return claimItem(opts, args[0], cmd.OutOrStdout(), true)
		},
	}
	cmd.Flags().BoolVar(&opts.dry, "dry-run", false, "Show transition without writing")
	return cmd
}

var errChecksFailed = errors.New("verification checks failed")

func newVerifyCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify <id>",
		Short: "Run the item's checks and record the outcome",
		Long: `Moves the item to verifying, runs each declared check command from the
repository root via a shell, then marks the item verified (all checks pass) or
failed (first failing check wins; non-zero exit). Evidence entries and per-check
logs under .factory/run/ record exactly what ran.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			item, err := opts.load(args[0])
			if err != nil {
				return err
			}
			if opts.dry {
				fmt.Fprintf(cmd.OutOrStdout(), "dry-run: would run %d check(s) for %s (state=%s):\n",
					len(item.Checks), item.ID, item.State)
				for _, c := range item.Checks {
					fmt.Fprintf(cmd.OutOrStdout(), "  $ %s\n", c)
				}
				return nil
			}
			switch item.State {
			case StateDelivered:
				fmt.Fprintf(cmd.OutOrStdout(), "%s already delivered; verification is a no-op\n", item.ID)
				return nil
			case StateInProgress, StateVerifying, StateVerified, StateFailed:
			default:
				return fmt.Errorf("item %s: cannot verify from state %s", item.ID, item.State)
			}
			if err := item.Transition(StateVerifying, opts.now()); err != nil {
				return err
			}
			if err := opts.save(item); err != nil {
				return err
			}
			runner := opts.runner()
			errOut := cmd.ErrOrStderr()
			runner.OnResult = func(res CheckResult) {
				status := "pass"
				if !res.OK() {
					status = fmt.Sprintf("FAIL exit=%d", res.ExitCode)
				}
				fmt.Fprintf(errOut, "check %d/%d %s: %s\n", res.Index+1, res.Total, status, res.Command)
			}
			results, err := runner.RunChecks(cmd.Context(), item)
			if err != nil {
				return err
			}
			var firstFail *CheckResult
			for i := range results {
				res := results[i]
				note := "pass"
				if !res.OK() {
					note = "fail"
				}
				item.Record(Evidence{
					Event:    "check",
					Command:  res.Command,
					ExitCode: res.ExitCode,
					Note:     note,
					Head:     opts.headFunc()(),
					Log:      res.Log,
				}, opts.now())
				if !res.OK() && firstFail == nil {
					firstFail = &results[i]
				}
			}
			if firstFail == nil {
				if err := item.Transition(StateVerified, opts.now()); err != nil {
					return err
				}
				item.LastError = ""
				item.Verified = &Verified{
					At:          opts.now().UTC().Format(time.RFC3339),
					Head:        opts.headFunc()(),
					Fingerprint: item.AcceptanceFingerprint(),
				}
				item.Record(Evidence{Event: "verify-pass", Head: opts.headFunc()()}, opts.now())
				if err := opts.save(item); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s verified: %d/%d checks passed\n", item.ID, len(results), len(item.Checks))
				return nil
			}
			errMsg := fmt.Sprintf("check failed (exit %d): %s", firstFail.ExitCode, firstFail.Command)
			item.LastError = errMsg
			if terr := item.Transition(StateFailed, opts.now()); terr != nil {
				return terr
			}
			item.Record(Evidence{
				Event: "verify-fail",
				Note:  errMsg,
				Head:  opts.headFunc()(),
			}, opts.now())
			if err := opts.save(item); err != nil {
				return err
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "%s failed: %s\n", item.ID, errMsg)
			return errChecksFailed
		},
	}
	cmd.Flags().BoolVar(&opts.dry, "dry-run", false, "Print checks without running or changing state")
	return cmd
}

func newDeliverCommand(opts *options) *cobra.Command {
	var note string
	cmd := &cobra.Command{
		Use:   "deliver <id>",
		Short: "Mark a verified item delivered",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			item, err := opts.load(args[0])
			if err != nil {
				return err
			}
			if item.State == StateDelivered {
				fmt.Fprintf(cmd.OutOrStdout(), "%s already delivered\n", item.ID)
				return nil
			}
			if item.State != StateVerified {
				return fmt.Errorf("item %s: deliver requires verified state (state=%s); run 'factory verify %s' first", item.ID, item.State, item.ID)
			}
			if item.Verified == nil || item.Verified.Fingerprint != item.AcceptanceFingerprint() {
				errMsg := "acceptance criteria or checks changed after verification; run 'factory verify " + item.ID + "' again"
				if terr := item.Transition(StateInProgress, opts.now()); terr != nil {
					return terr
				}
				item.LastError = errMsg
				item.Verified = nil
				item.Record(Evidence{Event: "stale-verify", Note: errMsg, Head: opts.headFunc()()}, opts.now())
				if !opts.dry {
					if serr := opts.save(item); serr != nil {
						return serr
					}
				}
				return fmt.Errorf("item %s: %s", item.ID, errMsg)
			}
			if err := item.Transition(StateDelivered, opts.now()); err != nil {
				return err
			}
			item.Record(Evidence{Event: "deliver", Note: note, Head: opts.headFunc()()}, opts.now())
			if opts.dry {
				fmt.Fprintf(cmd.OutOrStdout(), "dry-run: would deliver %s\n", item.ID)
				return nil
			}
			if err := opts.save(item); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "delivered %s\n", item.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&note, "note", "", "Delivery note (e.g. branch or commit)")
	cmd.Flags().BoolVar(&opts.dry, "dry-run", false, "Show transition without writing")
	return cmd
}

func newBlockCommand(opts *options) *cobra.Command {
	var reason string
	cmd := &cobra.Command{
		Use:   "block <id> --reason R",
		Short: "Park an item as blocked with a reason",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(reason) == "" {
				return fmt.Errorf("--reason is required")
			}
			item, err := opts.load(args[0])
			if err != nil {
				return err
			}
			if item.State == StateBlocked {
				fmt.Fprintf(cmd.OutOrStdout(), "%s already blocked: %s\n", item.ID, item.LastError)
				return nil
			}
			if err := item.Transition(StateBlocked, opts.now()); err != nil {
				return err
			}
			item.LastError = reason
			item.Record(Evidence{Event: "block", Note: reason, Head: opts.headFunc()()}, opts.now())
			if opts.dry {
				fmt.Fprintf(cmd.OutOrStdout(), "dry-run: would block %s\n", item.ID)
				return nil
			}
			if err := opts.save(item); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "blocked %s: %s\n", item.ID, reason)
			return nil
		},
	}
	cmd.Flags().StringVar(&reason, "reason", "", "Why the item is blocked")
	cmd.Flags().BoolVar(&opts.dry, "dry-run", false, "Show transition without writing")
	return cmd
}

func newUnblockCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "unblock <id>",
		Short: "Return a blocked item to the queue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			item, err := opts.load(args[0])
			if err != nil {
				return err
			}
			if item.State != StateBlocked {
				return fmt.Errorf("item %s: unblock requires blocked state (state=%s)", item.ID, item.State)
			}
			if err := item.Transition(StateQueued, opts.now()); err != nil {
				return err
			}
			item.LastError = ""
			item.Record(Evidence{Event: "unblock", Head: opts.headFunc()()}, opts.now())
			if err := opts.save(item); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "unblocked %s -> queued\n", item.ID)
			return nil
		},
	}
}

func newEvidenceCommand(opts *options) *cobra.Command {
	var note string
	cmd := &cobra.Command{
		Use:   "evidence <id> --note N",
		Short: "Append a manual evidence entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(note) == "" {
				return fmt.Errorf("--note is required")
			}
			item, err := opts.load(args[0])
			if err != nil {
				return err
			}
			item.Record(Evidence{Event: "note", Note: note, Head: opts.headFunc()()}, opts.now())
			if err := opts.save(item); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "noted %s\n", item.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&note, "note", "", "Evidence note to record")
	return cmd
}

func newStatusCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Summarize backlog counts by state",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			items, err := opts.store().List()
			if err != nil {
				return err
			}
			counts := map[State]int{}
			for _, it := range items {
				counts[it.State]++
			}
			if opts.json {
				return printJSON(cmd.OutOrStdout(), map[string]any{
					"total":  len(items),
					"counts": counts,
					"items":  items,
				})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "items=%d", len(items))
			for _, s := range States() {
				if counts[s] > 0 {
					fmt.Fprintf(cmd.OutOrStdout(), " %s=%d", s, counts[s])
				}
			}
			fmt.Fprintln(cmd.OutOrStdout())
			return nil
		},
	}
}
