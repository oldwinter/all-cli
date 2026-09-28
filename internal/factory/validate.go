package factory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/spf13/cobra"
)

// SchemaPath is the canonical location of the backlog item JSON schema.
const SchemaPath = ".factory/work-item.schema.json"

const itemSchemaURL = "https://github.com/oldwinter/all-cli/.factory/work-item.schema.json"

// ValidateBacklog compiles .factory/work-item.schema.json and validates every
// *.json file in the backlog directory against it plus WorkItem.Validate.
func ValidateBacklog(root string) error {
	schemaFile := filepath.Join(root, filepath.FromSlash(SchemaPath))
	raw, err := os.ReadFile(schemaFile)
	if err != nil {
		return fmt.Errorf("read %s: %w", SchemaPath, err)
	}
	var doc any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("parse %s: %w", SchemaPath, err)
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource(itemSchemaURL, doc); err != nil {
		return fmt.Errorf("load %s: %w", SchemaPath, err)
	}
	sch, err := c.Compile(itemSchemaURL)
	if err != nil {
		return fmt.Errorf("compile %s: %w", SchemaPath, err)
	}

	backlogDir := filepath.Join(root, ".factory", "backlog")
	entries, err := os.ReadDir(backlogDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read backlog %s: %w", backlogDir, err)
	}
	var failures []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".json") || strings.HasPrefix(name, ".tmp-") {
			continue
		}
		path := filepath.Join(backlogDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		var instance any
		if err := json.Unmarshal(data, &instance); err != nil {
			failures = append(failures, fmt.Sprintf("%s: invalid JSON: %v", name, err))
			continue
		}
		if err := sch.Validate(instance); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		var item WorkItem
		if err := json.Unmarshal(data, &item); err != nil {
			failures = append(failures, fmt.Sprintf("%s: decode: %v", name, err))
			continue
		}
		if err := item.Validate(); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		failures = append(failures, verifiedConsistencyFailures(name, &item)...)
	}
	if len(failures) > 0 {
		sort.Strings(failures)
		return fmt.Errorf("backlog validation failed:\n  %s", strings.Join(failures, "\n  "))
	}
	return nil
}

func newValidateCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate every backlog item against the JSON schema",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := ValidateBacklog(opts.root); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "backlog valid (%s)\n", filepath.Join(opts.root, ".factory", "backlog"))
			return nil
		},
	}
}

// verifiedConsistencyFailures enforces the verified-metadata lifecycle: a
// verified item must carry matching verification evidence, delivered keeps
// its record as provenance, and no other state may carry one.
func verifiedConsistencyFailures(name string, item *WorkItem) []string {
	if item.State == StateVerified {
		switch {
		case item.Verified == nil:
			return []string{fmt.Sprintf("%s: state=verified without verified metadata", name)}
		case item.Verified.Fingerprint != item.AcceptanceFingerprint():
			return []string{fmt.Sprintf("%s: verified fingerprint does not match current acceptance/checks", name)}
		}
		return nil
	}
	if item.State != StateDelivered && item.Verified != nil {
		return []string{fmt.Sprintf("%s: verified metadata on state=%s", name, item.State)}
	}
	return nil
}
