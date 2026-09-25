package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
	"github.com/oldwinter/all-cli/internal/tools"
)

func TestSchemaCommandGuidesWhenNameMissing(t *testing.T) {
	_, _, err := executeTestCommand(t, newSchemaCommand())
	if err == nil || !strings.Contains(err.Error(), "want schema status|diagnostic") || !strings.Contains(err.Error(), "all-cli schema status") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSchemaCommandStillRejectsUnknownName(t *testing.T) {
	_, _, err := executeTestCommand(t, newSchemaCommand(), "bogus")
	if err == nil || !strings.Contains(err.Error(), "invalid argument") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDescribeCommandPointsAtCatalogWhenIDMissing(t *testing.T) {
	_, _, err := executeTestCommand(t, newDescribeCommand(&rootOptions{}))
	if err == nil || !strings.Contains(err.Error(), "describe needs a tool ID") || !strings.Contains(err.Error(), "all-cli catalog") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestKubectlUseGuidesToList(t *testing.T) {
	_, _, err := executeTestCommand(t, newKubectlCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}), "use")
	if err == nil || !strings.Contains(err.Error(), "want kubectl use <context>") || !strings.Contains(err.Error(), "all-cli kubectl list") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDockerUseGuidesToList(t *testing.T) {
	_, _, err := executeTestCommand(t, newDockerCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}), "use")
	if err == nil || !strings.Contains(err.Error(), "want docker use <context>") || !strings.Contains(err.Error(), "all-cli docker list") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGLabUseGuidesToList(t *testing.T) {
	_, _, err := executeTestCommand(t, newGLabCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}), "use")
	if err == nil || !strings.Contains(err.Error(), "want glab use <host>") || !strings.Contains(err.Error(), "all-cli glab list") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestArgoCDUseGuidesToList(t *testing.T) {
	_, _, err := executeTestCommand(t, newArgoCDCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}), "use")
	if err == nil || !strings.Contains(err.Error(), "want argocd use <context>") || !strings.Contains(err.Error(), "all-cli argocd list") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestKargoUsePointsAtCurrent(t *testing.T) {
	_, _, err := executeTestCommand(t, newKargoCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}), "use")
	if err == nil || !strings.Contains(err.Error(), "project name is required") || !strings.Contains(err.Error(), "all-cli kargo current") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGHUseGuidesToList(t *testing.T) {
	for _, args := range [][]string{
		{"use"},
		{"use", "--hostname", "github.com"},
	} {
		_, _, err := executeTestCommand(t, newGHCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}), args...)
		if err == nil || !strings.Contains(err.Error(), "want gh use --hostname <host> --user <login>") || !strings.Contains(err.Error(), "all-cli gh list") {
			t.Fatalf("args %v: unexpected error: %v", args, err)
		}
	}
}

func TestGHUseHelpShowsListExample(t *testing.T) {
	stdout, _, err := executeTestCommand(t, newGHCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}), "use", "--help")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "all-cli gh list") {
		t.Fatalf("expected list example in help, got:\n%s", stdout)
	}
}

func TestDiffGuidesToSnapshotWhenArgsMissing(t *testing.T) {
	for _, args := range [][]string{{}, {"before.json"}, {"a.json", "b.json", "c.json"}} {
		_, _, err := executeTestCommand(t, newDiffCommand(&rootOptions{}), args...)
		if err == nil || !strings.Contains(err.Error(), "want diff <snapshot-a> <snapshot-b>") || !strings.Contains(err.Error(), "all-cli snapshot --json") {
			t.Fatalf("args %v: unexpected error: %v", args, err)
		}
	}
}

func TestFixRequiresDryRunWithExample(t *testing.T) {
	_, _, err := executeTestCommand(t, newFixCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{}))
	if err == nil || !strings.Contains(err.Error(), "fix currently requires --dry-run") || !strings.Contains(err.Error(), "all-cli fix --dry-run") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestKubectlCurrentEmptyGuidesToStatus(t *testing.T) {
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"kubectl config current-context": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "context missing",
			},
			"kubectl config view --minify --output jsonpath={..namespace}{\"\\n\"}": {
				Stdout: "",
			},
		},
	}

	stdout, stderr, err := executeTestCommand(t, newKubectlCommand(&rootOptions{Timeout: time.Second}, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "See install status: all-cli kubectl status") {
		t.Fatalf("expected status hint in stdout, got:\n%s", stdout)
	}
	if !strings.Contains(stderr, "error: kubectl config current-context failed") {
		t.Fatalf("expected error on stderr, got %q", stderr)
	}
}

func TestK9sCurrentEmptyChecksStatus(t *testing.T) {
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"kubectl config current-context": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "context missing",
			},
			"kubectl config view --minify --output jsonpath={..namespace}{\"\\n\"}": {Stdout: ""},
			"k9s info": {
				ExitCode: 1,
				Err:      assertError("exit status 1"),
				Stderr:   "no k9s config",
			},
		},
	}

	_, stderr, err := executeTestCommand(t, newK9sCommand(&rootOptions{Timeout: time.Second}, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stderr, "no k9s context. Check: all-cli k9s status") {
		t.Fatalf("expected k9s status hint on stderr, got %q", stderr)
	}
}

func TestMiseCurrentEmptyGuidesToStatus(t *testing.T) {
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"mise current": {Stdout: "\n"},
		},
	}

	stdout, _, err := executeTestCommand(t, newMiseCommand(&rootOptions{Timeout: time.Second}, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "See install status: all-cli mise status") {
		t.Fatalf("expected mise status hint, got:\n%s", stdout)
	}
}

func TestAliyunCurrentEmptyGuidesToListAndStatus(t *testing.T) {
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"aliyun configure list": {Stdout: "\n"},
		},
	}

	stdout, _, err := executeTestCommand(t, newAliyunCommand(&rootOptions{Timeout: time.Second}, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "all-cli aliyun list") || !strings.Contains(stdout, "all-cli aliyun status") {
		t.Fatalf("expected aliyun list/status hint, got:\n%s", stdout)
	}
}

func TestWranglerCurrentLoggedOutGuidesToStatus(t *testing.T) {
	runner := cliFakeRunner{
		results: map[string]execx.CmdResult{
			"wrangler whoami --json": {
				Stdout: `{"loggedIn":false,"accounts":[]}`,
			},
		},
	}

	stdout, _, err := executeTestCommand(t, newWranglerCommand(&rootOptions{Timeout: time.Second}, runner), "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "See install and login status: all-cli wrangler status") {
		t.Fatalf("expected wrangler status hint, got:\n%s", stdout)
	}
}

func TestPrintDiagnosticReportEmptySuggestsNextSteps(t *testing.T) {
	var out strings.Builder
	printDiagnosticReport(&out, model.DiagnosticReport{})
	got := out.String()
	for _, needle := range []string{
		"No diagnostics found.",
		"all-cli catalog",
		"all-cli status",
		"all-cli fix --dry-run",
	} {
		if !strings.Contains(got, needle) {
			t.Fatalf("expected %q in empty diagnostics output, got:\n%s", needle, got)
		}
	}
}

func TestDoctorAndDiagnoseHelpDifferentiate(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	runner := cliFakeRunner{}
	doctor := newDoctorCommand(opts, runner)
	diagnose := newDiagnoseCommand(opts, runner)

	if doctor.Short == diagnose.Short {
		t.Fatalf("doctor and diagnose shorts must differ: %q", doctor.Short)
	}
	if !strings.Contains(doctor.Long, "diagnose") {
		t.Fatalf("doctor long should point at diagnose, got: %q", doctor.Long)
	}
	if !strings.Contains(diagnose.Long, "doctor") {
		t.Fatalf("diagnose long should point at doctor, got: %q", diagnose.Long)
	}
}

func TestCurrentLongHasNoStrayTab(t *testing.T) {
	current := newCurrentCommand(&rootOptions{Timeout: time.Second}, cliFakeRunner{})
	if strings.Contains(current.Long, "\t") {
		t.Fatalf("current long contains a tab: %q", current.Long)
	}
}

func TestOptionsCommandListsNoProgress(t *testing.T) {
	opts := &rootOptions{Timeout: time.Second}
	stdout, _, err := executeTestCommand(t, newOptionsCommand(opts))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "no_progress=false") {
		t.Fatalf("expected no_progress=false in output, got:\n%s", stdout)
	}
}

func TestOptionsCommandJSONListsNoProgress(t *testing.T) {
	opts := &rootOptions{JSON: true, NoProgress: true, Timeout: time.Second}
	stdout, _, err := executeTestCommand(t, newOptionsCommand(opts))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, `"no_progress": true`) {
		t.Fatalf("expected no_progress in JSON output, got:\n%s", stdout)
	}
}

func TestRootHelpExampleIncludesCatalog(t *testing.T) {
	if !strings.Contains(NewRootCommand().Example, "all-cli catalog") {
		t.Fatal("root example should include all-cli catalog")
	}
}

func TestSnapshotHelpExplainsJSONForDiff(t *testing.T) {
	snapshot := newSnapshotCommand(&rootOptions{}, cliFakeRunner{})
	if !strings.Contains(snapshot.Long, "--json") || !strings.Contains(snapshot.Long, "diff") {
		t.Fatalf("snapshot long should explain --json for diff, got: %q", snapshot.Long)
	}
	if !strings.Contains(snapshot.Example, "all-cli snapshot --json") {
		t.Fatalf("snapshot example should show --json, got: %q", snapshot.Example)
	}
}

func TestSurpriseSignatureStaysEnglish(t *testing.T) {
	stdout, _, err := executeTestCommand(t, newSurpriseCommand())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "fortune favors the curious") {
		t.Fatalf("expected English signature, got:\n%s", stdout)
	}
	if strings.Contains(stdout, "好奇") {
		t.Fatalf("unexpected Chinese signature line, got:\n%s", stdout)
	}
}

func TestCatalogTracksGrok(t *testing.T) {
	if _, ok := tools.FindByID("grok"); !ok {
		t.Fatal("expected grok in the tracked tool registry")
	}
	stdout, _, err := executeTestCommand(t, newCatalogCommand(&rootOptions{}), "grok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "grok") {
		t.Fatalf("expected grok in catalog search, got:\n%s", stdout)
	}
}
