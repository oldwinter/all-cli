package aliyun

import "testing"

func TestParseConfigureListKeepsProfileNamesWithHeaderPrefix(t *testing.T) {
	stdout := `Profile   | Credential | Valid | Region | Language
--------- | ---------- | ----- | ------ | --------
ProfileProd | AK:*** | Valid | us-east-1 | en
`
	profiles, warnings, errs, err := parseConfigureList(stdout)
	if err != nil || len(warnings) != 0 || len(errs) != 0 {
		t.Fatalf("parseConfigureList() diagnostics = %#v %#v %v", warnings, errs, err)
	}
	if len(profiles) != 1 || profiles[0].Name != "ProfileProd" {
		t.Fatalf("profile row sharing the header prefix was dropped: %#v", profiles)
	}
}
