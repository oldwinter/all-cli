package docker

import "testing"

func TestParseContextLSJSONLines(t *testing.T) {
	stdout := `{"Current":false,"Description":"Current DOCKER_HOST based configuration","DockerEndpoint":"unix:///var/run/docker.sock","Error":"","Name":"default"}
{"Current":true,"Description":"","DockerEndpoint":"ssh://cdd@192.168.10.118","Error":"","Name":"remote118"}
`
	contexts, warnings, errs, err := parseContextLSJSONLines(stdout)
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
	if contexts[0].Name != "default" || contexts[0].IsCurrent {
		t.Fatalf("unexpected first context: %#v", contexts[0])
	}
	if contexts[1].Name != "remote118" || !contexts[1].IsCurrent {
		t.Fatalf("unexpected second context: %#v", contexts[1])
	}
}

func TestParseContextLSJSONLines_InvalidLine(t *testing.T) {
	stdout := `not json
{"Current":true,"Description":"","DockerEndpoint":"ssh://cdd@192.168.10.118","Error":"","Name":"remote118"}
`
	contexts, _, errs, err := parseContextLSJSONLines(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) == 0 {
		t.Fatalf("expected errs for invalid JSON line")
	}
	if len(contexts) != 1 || contexts[0].Name != "remote118" {
		t.Fatalf("unexpected contexts: %#v", contexts)
	}
}
