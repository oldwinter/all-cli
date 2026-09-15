package cli

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/oldwinter/all-cli/internal/model"
)

func TestDescribeCommandPrintsReadOnlyExamples(t *testing.T) {
	for _, tc := range []struct {
		tool        string
		wantCurrent bool
	}{
		{tool: "kubectl", wantCurrent: true},
		{tool: "aws", wantCurrent: true},
		{tool: "fd", wantCurrent: false},
	} {
		t.Run(tc.tool, func(t *testing.T) {
			stdout, stderr, err := executeTestCommand(t, newDescribeCommand(&rootOptions{}), tc.tool)
			if err != nil || stderr != "" {
				t.Fatalf("describe: err=%v stderr=%q", err, stderr)
			}
			_, examples, found := strings.Cut(stdout, "Examples:\n")
			want := "  all-cli status --tools " + tc.tool + "\n  all-cli doctor --tools " + tc.tool + "\n"
			if tc.wantCurrent {
				want += "  all-cli current --tools " + tc.tool + "\n"
			}
			if !found || examples != want {
				t.Fatalf("examples = %q, want %q", examples, want)
			}
		})
	}
}

func TestDescribeJSONKeepsMetadataOnly(t *testing.T) {
	stdout, _, err := executeTestCommand(t, newDescribeCommand(&rootOptions{JSON: true}), "kubectl")
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(stdout), &fields); err != nil {
		t.Fatal(err)
	}
	wantKeys := []string{"id", "display_name", "category", "binary", "capabilities", "metadata"}
	if len(fields) != len(wantKeys) {
		t.Fatalf("unexpected JSON fields: %v", fields)
	}
	for _, key := range wantKeys {
		if _, ok := fields[key]; !ok {
			t.Errorf("missing JSON field %q", key)
		}
	}
}

type exampleErrorWriter struct {
	err error
}

func (w exampleErrorWriter) Write(p []byte) (int, error) {
	if strings.Contains(string(p), "Examples:") {
		return 0, w.err
	}
	return len(p), nil
}

func TestDescribeCommandReturnsExampleWriteError(t *testing.T) {
	want := errors.New("output closed")
	cmd := newDescribeCommand(&rootOptions{})
	cmd.SetOut(exampleErrorWriter{err: want})
	if err := cmd.RunE(cmd, []string{"fd"}); !errors.Is(err, want) {
		t.Fatalf("error = %v, want %v", err, want)
	}
}

func TestDescribeCommandPrintsHumanReadableMetadata(t *testing.T) {
	// Given
	opts := &rootOptions{}

	// When
	stdout, stderr, err := executeTestCommand(t, newDescribeCommand(opts), "kubectl")

	// Then
	if err != nil {
		t.Fatalf("describe kubectl: %v", err)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"Tool: kubectl",
		"Category: k8s",
		"Binary: kubectl",
		"Purpose:",
		"Configured when:",
		"Has contexts: yes",
		"Can switch: yes",
		"Current fields:",
		"  context:",
		"Agent actions:",
		"  - inspect_status",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestDescribeCommandPrintsJSONMetadata(t *testing.T) {
	// Given
	opts := &rootOptions{JSON: true}

	// When
	stdout, stderr, err := executeTestCommand(t, newDescribeCommand(opts), "kubectl")

	// Then
	if err != nil {
		t.Fatalf("describe kubectl --json: %v", err)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	var got struct {
		ID           string             `json:"id"`
		DisplayName  string             `json:"display_name"`
		Category     string             `json:"category"`
		Binary       string             `json:"binary"`
		Capabilities model.Capability   `json:"capabilities"`
		Metadata     model.ToolMetadata `json:"metadata"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode describe json: %v", err)
	}
	if got.ID != "kubectl" || got.DisplayName != "kubectl" || got.Category != "k8s" || got.Binary != "kubectl" {
		t.Fatalf("unexpected identity: %#v", got)
	}
	if !got.Capabilities.HasContexts || !got.Capabilities.CanSwitch {
		t.Fatalf("unexpected capabilities: %#v", got.Capabilities)
	}
	if got.Metadata.Purpose == "" || got.Metadata.ConfiguredWhen == "" || len(got.Metadata.AgentActions) == 0 {
		t.Fatalf("unexpected metadata: %#v", got.Metadata)
	}
}

func TestDescribeCommandRejectsUnknownTool(t *testing.T) {
	// Given
	opts := &rootOptions{}

	// When
	_, _, err := executeTestCommand(t, newDescribeCommand(opts), "not-a-tool")

	// Then
	if err == nil || !strings.Contains(err.Error(), `unknown tool ID "not-a-tool"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDescribeCommandEscapesUnknownToolID(t *testing.T) {
	// Given
	opts := &rootOptions{}
	toolID := "bad\x1b[31m"

	// When
	_, _, err := executeTestCommand(t, newDescribeCommand(opts), toolID)

	// Then
	if err == nil {
		t.Fatal("expected unknown tool error")
	}
	if strings.Contains(err.Error(), "\x1b") {
		t.Fatalf("error contains a terminal escape: %q", err)
	}
	if !strings.Contains(err.Error(), strconv.Quote(toolID)) {
		t.Fatalf("error does not quote the tool ID: %q", err)
	}
}

func TestDescribeCommandEscapesFlagParserErrors(t *testing.T) {
	// Given
	root := NewRootCommand()
	toolID := "--bad\x1b[31m"

	// When
	_, _, err := executeTestCommand(t, root, "describe", toolID)

	// Then
	if err == nil {
		t.Fatal("expected flag parser error")
	}
	if strings.Contains(err.Error(), "\x1b") {
		t.Fatalf("error contains a terminal escape: %q", err)
	}
	if !strings.Contains(err.Error(), `\x1b`) {
		t.Fatalf("error does not escape the terminal control: %q", err)
	}
}

func TestDescribeCommandCompletesToolIDs(t *testing.T) {
	// Given
	cmd := newDescribeCommand(&rootOptions{})

	// When
	got, directive := cmd.ValidArgsFunction(cmd, nil, "kube")

	// Then
	if directive == 0 {
		t.Fatal("expected a completion directive")
	}
	found := false
	for _, candidate := range got {
		if strings.HasPrefix(candidate, "kubectl\t") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("kubectl completion missing: %#v", got)
	}
}

func TestRootCommandIncludesDescribe(t *testing.T) {
	// Given
	root := NewRootCommand()

	// When
	cmd, _, err := root.Find([]string{"describe"})

	// Then
	if err != nil {
		t.Fatalf("find describe command: %v", err)
	}
	if cmd.Name() != "describe" {
		t.Fatalf("command name = %q, want describe", cmd.Name())
	}
}
