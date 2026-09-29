package argocd

import "testing"

func TestParseContextTableKeepsNamesWithHeaderPrefix(t *testing.T) {
	contexts, warnings, errs, err := parseContextTable("CURRENT NAME SERVER\nCURRENTLY https://argo.example\n")
	if err != nil || len(warnings) != 0 || len(errs) != 0 {
		t.Fatalf("parseContextTable() diagnostics = %#v %#v %v", warnings, errs, err)
	}
	if len(contexts) != 1 || contexts[0].Name != "CURRENTLY" {
		t.Fatalf("context row sharing the header prefix was dropped: %#v", contexts)
	}
}
