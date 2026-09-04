package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/oldwinter/all-cli/schemas"
)

func TestSchemaCommandPrintsBundledSchemas(t *testing.T) {
	t.Parallel()

	for _, name := range schemas.Names() {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stdout, stderr, err := executeTestCommand(t, newSchemaCommand(), name)
			if err != nil {
				t.Fatalf("schema %s: %v", name, err)
			}
			if stderr != "" {
				t.Fatalf("stderr = %q, want empty", stderr)
			}
			want, err := schemas.Read(name)
			if err != nil {
				t.Fatalf("read expected schema: %v", err)
			}
			if stdout != string(want) {
				t.Fatalf("schema output differs from bundled %s schema", name)
			}
			var decoded map[string]any
			if err := json.Unmarshal([]byte(stdout), &decoded); err != nil {
				t.Fatalf("decode schema output: %v", err)
			}
		})
	}
}

func TestSchemaCommandRejectsUnknownName(t *testing.T) {
	t.Parallel()

	_, _, err := executeTestCommand(t, newSchemaCommand(), "other")
	if err == nil || !strings.Contains(err.Error(), "invalid argument \"other\"") {
		t.Fatalf("schema other error = %v, want invalid argument", err)
	}
}

func TestRootCommandIncludesSchema(t *testing.T) {
	t.Parallel()

	root := NewRootCommand()
	schema, _, err := root.Find([]string{"schema"})
	if err != nil {
		t.Fatalf("find schema command: %v", err)
	}
	if schema.Name() != "schema" || schema.GroupID != "primary" {
		t.Fatalf("unexpected schema command registration: name=%q group=%q", schema.Name(), schema.GroupID)
	}
}
