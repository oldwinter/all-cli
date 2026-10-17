package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/oldwinter/all-cli/internal/model"
)

func printSnapshotChangeDetails(w io.Writer, change model.SnapshotToolChange) error {
	if len(change.Fields) == 0 {
		return nil
	}
	before, err := snapshotDetailFields(change.Before)
	if err != nil {
		return err
	}
	after, err := snapshotDetailFields(change.After)
	if err != nil {
		return err
	}
	for _, field := range change.Fields {
		if _, err := fmt.Fprintf(w, "    %s: %s -> %s\n", field,
			snapshotDetailValue(before, field), snapshotDetailValue(after, field)); err != nil {
			return err
		}
	}
	return nil
}

func snapshotDetailFields(tool *model.ToolSummary) (map[string]json.RawMessage, error) {
	data, err := json.Marshal(tool)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	return fields, nil
}

func snapshotDetailValue(fields map[string]json.RawMessage, field string) string {
	if value, ok := fields[field]; ok {
		return string(value)
	}
	return "null"
}
