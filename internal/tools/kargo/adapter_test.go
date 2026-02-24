package kargo

import "testing"

func TestParseConfigView(t *testing.T) {
	stdout := `apiAddress: http://192.168.139.2:8444
bearerToken: '*** REDACTED ***'
insecureSkipTLSVerify: true
kind: CLIConfig
refreshToken: '*** REDACTED ***'
defaultProject: demo
`

	cfg, warnings, errs, err := parseConfigView(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %#v", errs)
	}
	if cfg.APIAddress != "http://192.168.139.2:8444" {
		t.Fatalf("unexpected api address: %q", cfg.APIAddress)
	}
	if cfg.DefaultProject != "demo" {
		t.Fatalf("unexpected default project: %q", cfg.DefaultProject)
	}
}
