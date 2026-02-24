package aliyun

import "testing"

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
