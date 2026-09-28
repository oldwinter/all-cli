package aliyun

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/oldwinter/all-cli/internal/execx"
)

type fakeRunner struct {
	results map[string]execx.CmdResult
}

func (f fakeRunner) Run(_ context.Context, name string, args ...string) execx.CmdResult {
	key := name
	if len(args) > 0 {
		key += " " + strings.Join(args, " ")
	}
	if res, ok := f.results[key]; ok {
		return res
	}
	return execx.CmdResult{ExitCode: 1, Err: errors.New("unexpected command"), Stderr: "unexpected command"}
}

func TestParseConfigureList(t *testing.T) {
	stdout := `Profile   | Credential         | Valid   | Region           | Language
--------- | ------------------ | ------- | ---------------- | --------
default * | AK:***6ps          | Valid   | cn-hangzhou      | zh
dev       | AK:***123          | Invalid | us-east-1        | en
`

	profiles, warnings, errs, err := parseConfigureList(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %#v", errs)
	}
	if len(profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(profiles))
	}
	if profiles[0].Name != "default" || !profiles[0].IsCurrent {
		t.Fatalf("unexpected first profile: %#v", profiles[0])
	}
	if profiles[0].Region != "cn-hangzhou" || profiles[0].Language != "zh" || profiles[0].Valid != "Valid" {
		t.Fatalf("unexpected first profile fields: %#v", profiles[0])
	}
	if profiles[1].Name != "dev" || profiles[1].IsCurrent {
		t.Fatalf("unexpected second profile: %#v", profiles[1])
	}
}

func TestAdapterCurrent_FallsBackToFirstProfile(t *testing.T) {
	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"aliyun configure list": {
				Stdout: `Profile   | Credential         | Valid   | Region      | Language
--------- | ------------------ | ------- | ----------- | --------
default   | AK:***6ps          | Valid   | cn-hangzhou | zh
dev       | AK:***123          | Invalid | us-east-1   | en
`,
			},
		},
	})

	cur, warnings, errs, err := a.Current(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %#v", errs)
	}
	if cur["profile"] != "default" || cur["region"] != "cn-hangzhou" {
		t.Fatalf("unexpected current: %#v", cur)
	}
	if len(warnings) != 1 || warnings[0] != "no current aliyun profile marked; using the first profile" {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
}

func TestAdapterCurrent_UsesStarMarkedProfile(t *testing.T) {
	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"aliyun configure list": {
				Stdout: `Profile   | Credential         | Valid   | Region      | Language
--------- | ------------------ | ------- | ----------- | --------
default   | AK:***6ps          | Valid   | cn-hangzhou | zh
* dev     | AK:***123          | Valid   | us-east-1   | en
`,
			},
		},
	})
	cur, warnings, errs, err := a.Current(context.Background())
	if err != nil || len(errs) != 0 {
		t.Fatalf("unexpected error: %v errs=%v", err, errs)
	}
	if cur["profile"] != "dev" || cur["region"] != "us-east-1" {
		t.Fatalf("star-marked profile must win: %#v", cur)
	}
	if len(warnings) != 0 {
		t.Fatalf("no warning expected when a profile is current: %#v", warnings)
	}
}

func TestParseConfigureListSkipsBlankLinesAndHugeLineFails(t *testing.T) {
	profiles, _, errs, err := parseConfigureList("\n\n* dev   | AK:***x | Valid | us-east-1 | en\n\n")
	if err != nil || len(errs) != 0 || len(profiles) != 1 {
		t.Fatalf("blank lines must be skipped: %v %#v %#v", profiles, errs, err)
	}
	if !profiles[0].IsCurrent {
		t.Fatalf("star-marked profile not flagged: %#v", profiles[0])
	}

	_, _, errs, err = parseConfigureList(strings.Repeat("x", 80*1024) + "\n")
	if err == nil || len(errs) == 0 {
		t.Fatal("expected scanner error on oversized line")
	}
}

func TestAdapterConfigured(t *testing.T) {
	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"aliyun configure list": {
				Stdout: `Profile   | Credential         | Valid   | Region      | Language
--------- | ------------------ | ------- | ----------- | --------
default * | AK:***6ps          | Valid   | cn-hangzhou | zh
`,
			},
		},
	})

	ok, warnings, errs, err := a.Configured(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected configured=true with profiles present")
	}
	if len(warnings) != 0 || len(errs) != 0 {
		t.Fatalf("unexpected diagnostics: %#v %#v", warnings, errs)
	}
}

func TestAdapterConfigured_Failure(t *testing.T) {
	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"aliyun configure list": {
				ExitCode: 1,
				Err:      errors.New("exit status 1"),
				Stderr:   "broken",
			},
		},
	})

	ok, _, errs, err := a.Configured(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if ok {
		t.Fatal("expected configured=false on failure")
	}
	if len(errs) != 1 || errs[0] != "broken" {
		t.Fatalf("unexpected errs: %#v", errs)
	}
}

func TestAdapterListProfilesSkipsBlankNamesAndNonTableLines(t *testing.T) {
	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"aliyun configure list": {
				Stdout: `some preamble
Profile   | Credential | Valid | Region      | Language
--------- | ---------- | ----- | ----------- | --------
*         | AK:***xx   | Valid | cn-beijing  | zh
prod      | AK:***yy   | Valid | us-west-1   | en
short | row
`,
			},
		},
	})

	profiles, warnings, errs, err := a.ListProfiles(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %#v", errs)
	}
	if len(profiles) != 1 || profiles[0].Name != "prod" || profiles[0].IsCurrent {
		t.Fatalf("unexpected profiles: %#v", profiles)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected short-row warning, got %#v", warnings)
	}
}

func TestParseConfigureList_WarnsOnShortRows(t *testing.T) {
	stdout := `Profile   | Credential         | Valid
--------- | ------------------ | -------
broken    | AK:***123          | Invalid
`

	profiles, warnings, errs, err := parseConfigureList(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(profiles) != 0 {
		t.Fatalf("expected no profiles, got %#v", profiles)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %#v", errs)
	}
	if len(warnings) != 1 || warnings[0] != "unexpected aliyun configure list output format" {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
}

func TestCurrentProfileFallbacks(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// ListProfiles failure propagates.
	a := New(fakeRunner{results: map[string]execx.CmdResult{
		"aliyun configure list": {Err: errors.New("boom"), ExitCode: 1, Stderr: "nope"},
	}})
	if _, _, errs, err := a.Current(ctx); err == nil || len(errs) != 1 || errs[0] != "nope" {
		t.Fatalf("err=%v errs=%v", err, errs)
	}

	// No profiles -> nil current.
	a = New(fakeRunner{results: map[string]execx.CmdResult{
		"aliyun configure list": {Stdout: "Profile   | Credential   | Valid   | Region   | Language\n--------- | ---------  | -----   | ------   | --------\n"},
	}})
	cur, _, _, err := a.Current(ctx)
	if err != nil || cur != nil {
		t.Fatalf("cur=%#v err=%v", cur, err)
	}

	// No starred profile -> warning + first profile; blank region/language omitted.
	a = New(fakeRunner{results: map[string]execx.CmdResult{
		"aliyun configure list": {Stdout: "prof1 | ak | Valid |  | \nprof2 | ak | Invalid | cn-hangzhou | en\n"},
	}})
	cur, warnings, _, err := a.Current(ctx)
	if err != nil || cur["profile"] != "prof1" {
		t.Fatalf("cur=%#v err=%v", cur, err)
	}
	if len(warnings) == 0 || !strings.Contains(warnings[0], "no current aliyun profile") {
		t.Fatalf("warnings = %#v", warnings)
	}
	if _, ok := cur["region"]; ok {
		t.Fatalf("expected blank region omitted: %#v", cur)
	}
}

func TestParseConfigureListSkipsBadLines(t *testing.T) {
	t.Parallel()
	profiles, warnings, _, err := parseConfigureList("no pipe line\na|b\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 0 {
		t.Fatalf("profiles = %#v", profiles)
	}
	if len(warnings) != 1 {
		t.Fatalf("warnings = %#v", warnings)
	}
}

func TestListProfilesBlankStderrFallsBackToErr(t *testing.T) {
	t.Parallel()
	a := New(fakeRunner{results: map[string]execx.CmdResult{
		"aliyun configure list": {Err: errors.New("spawn exploded"), ExitCode: 2},
	}})
	_, _, errs, err := a.ListProfiles(context.Background())
	if err == nil || len(errs) != 1 || errs[0] != "spawn exploded" {
		t.Fatalf("err=%v errs=%v", err, errs)
	}
}

func TestParseConfigureListSkipsBlankName(t *testing.T) {
	t.Parallel()
	profiles, _, _, err := parseConfigureList("   | ak | Valid | r | l\nok | ak | Valid | r | l\n")
	if err != nil || len(profiles) != 1 || profiles[0].Name != "ok" {
		t.Fatalf("profiles = %#v err=%v", profiles, err)
	}
}
