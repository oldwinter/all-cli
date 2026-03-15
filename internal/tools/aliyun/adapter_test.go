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
