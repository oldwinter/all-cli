package schemas

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestReadReturnsBundledSchemas(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		wantID string
	}{
		{name: Status, wantID: "status-report-v0.1.json"},
		{name: Diagnostic, wantID: "diagnostic-report-v0.1.json"},
		{name: FixPlan, wantID: "fix-plan-v0.1.json"},
		{name: SnapshotDiff, wantID: "snapshot-diff-v0.1.json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			content, err := Read(tt.name)
			if err != nil {
				t.Fatalf("Read(%q): %v", tt.name, err)
			}
			var schema struct {
				ID string `json:"$id"`
			}
			if err := json.Unmarshal(content, &schema); err != nil {
				t.Fatalf("decode %q schema: %v", tt.name, err)
			}
			if !strings.HasSuffix(schema.ID, "/"+tt.wantID) {
				t.Fatalf("schema ID = %q, want suffix %q", schema.ID, tt.wantID)
			}
		})
	}
}

func TestNamesIsStable(t *testing.T) {
	t.Parallel()

	want := []string{Status, Diagnostic, FixPlan, SnapshotDiff}
	got := Names()
	if len(got) != len(want) {
		t.Fatalf("Names() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Names() = %v, want %v", got, want)
		}
	}
}

// TestEverySchemaCompilesStandalone is the regression test for the offline
// validation promise: `all-cli schema <name> > file.json` must produce a file a
// consumer can compile on its own, with no network access and no sibling files.
// The diagnostic schema previously used a relative cross-file $ref and failed.
func TestEverySchemaCompilesStandalone(t *testing.T) {
	t.Parallel()

	for _, name := range Names() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			content, err := Read(name)
			if err != nil {
				t.Fatalf("Read(%q): %v", name, err)
			}

			doc, err := jsonschema.UnmarshalJSON(strings.NewReader(string(content)))
			if err != nil {
				t.Fatalf("unmarshal %q: %v", name, err)
			}

			compiler := jsonschema.NewCompiler()
			// Deliberately register nothing else: the schema must be
			// self-contained. Any relative $ref will now fail to resolve.
			compiler.UseLoader(noNetworkLoader{})

			var id struct {
				ID string `json:"$id"`
			}
			if err := json.Unmarshal(content, &id); err != nil {
				t.Fatalf("decode $id for %q: %v", name, err)
			}
			if err := compiler.AddResource(id.ID, doc); err != nil {
				t.Fatalf("add resource %q: %v", name, err)
			}
			if _, err := compiler.Compile(id.ID); err != nil {
				t.Fatalf("schema %q is not self-contained: %v", name, err)
			}
		})
	}
}

// noNetworkLoader makes any external $ref resolution fail loudly instead of
// silently reaching the network during tests.
type noNetworkLoader struct{}

func (noNetworkLoader) Load(url string) (any, error) {
	return nil, &unresolvableRefError{url: url}
}

type unresolvableRefError struct{ url string }

func (e *unresolvableRefError) Error() string {
	return "schema is not self-contained: tried to resolve external $ref " + e.url
}

func TestReadRejectsUnknownSchema(t *testing.T) {
	t.Parallel()

	_, err := Read("other")
	if err == nil || !strings.Contains(err.Error(), "expected one of:") {
		t.Fatalf("Read(other) error = %v, want supported names", err)
	}
}
