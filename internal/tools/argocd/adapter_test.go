package argocd

import "testing"

func TestParseContextTable(t *testing.T) {
	stdout := `CURRENT  NAME                  SERVER
         localhost:8080        localhost:8080
*        localhost:18443       localhost:18443
`

	contexts, warnings, errs, err := parseContextTable(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %#v", errs)
	}
	if len(contexts) != 2 {
		t.Fatalf("expected 2 contexts, got %d", len(contexts))
	}
	if contexts[0].IsCurrent {
		t.Fatalf("expected first context not current: %#v", contexts[0])
	}
	if !contexts[1].IsCurrent || contexts[1].Name != "localhost:18443" {
		t.Fatalf("unexpected current context: %#v", contexts[1])
	}
}
