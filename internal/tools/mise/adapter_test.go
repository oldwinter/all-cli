package mise

import "testing"

func TestParseMiseCurrent(t *testing.T) {
	stdout := `bun 1.3.0
go 1.26.0
node 22.20.0
python 3.14.0
`
	cur, warnings, errs, err := parseMiseCurrent(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %#v", errs)
	}
	if cur["go"] != "1.26.0" || cur["python"] != "3.14.0" {
		t.Fatalf("unexpected current map: %#v", cur)
	}
}
