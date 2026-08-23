package glab

import "testing"

func TestParseAuthStatusAll(t *testing.T) {
	stdout := `192.168.10.117:6001
  ✓ Logged in to 192.168.10.117:6001 as oldwinter (/Users/cdd/.config/glab-cli/config.yml)
  ✓ Git operations for 192.168.10.117:6001 configured to use ssh protocol.
  ✓ API calls for 192.168.10.117:6001 are made over http protocol.
  ✓ REST API Endpoint: http://192.168.10.117:6001/api/v4/
  ✓ GraphQL Endpoint: http://192.168.10.117:6001/api/graphql/
  ✓ Token found: **************************
http://bad-host:6001
  x http://bad-host:6001: API call failed: Get "https://http//bad-host:6001/api/v4/user": EOF
  ✓ API calls for http://bad-host:6001 are made over https protocol.
  ! No token found (checked config file, keyring, and environment variables).

   ERROR

  X could not authenticate to one or more of the configured GitLab instances..
`

	instances, warnings, errs, err := parseAuthStatusAll(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %#v", errs)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	if len(instances) != 2 {
		t.Fatalf("expected 2 instances, got %d", len(instances))
	}

	ok := instances[0]
	if ok.Host != "192.168.10.117:6001" || ok.User != "oldwinter" || !ok.OK || !ok.HasToken {
		t.Fatalf("unexpected ok instance: %#v", ok)
	}
	if ok.GitProtocol != "ssh" || ok.APIProtocol != "http" {
		t.Fatalf("unexpected protocols: %#v", ok)
	}
	if ok.RESTEndpoint == "" || ok.GraphQLEndpoint == "" {
		t.Fatalf("expected endpoints, got %#v", ok)
	}

	bad := instances[1]
	if bad.Host != "http://bad-host:6001" || bad.OK {
		t.Fatalf("unexpected bad instance: %#v", bad)
	}
	if bad.HasToken {
		t.Fatalf("expected has_token=false for bad instance: %#v", bad)
	}
	if bad.Error == "" {
		t.Fatalf("expected error for bad instance: %#v", bad)
	}
}
