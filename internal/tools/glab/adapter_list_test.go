package glab

import (
	"context"
	"strings"
	"testing"

	"github.com/oldwinter/all-cli/internal/execx"
)

type fakeRunner struct {
	results map[string]execx.CmdResult
}

func (f fakeRunner) Run(_ context.Context, name string, args ...string) execx.CmdResult {
	key := name
	if len(args) > 0 {
		key += " " + strings.Join(args, " ")
	}
	if res, ok := f.results[key]; ok {
		return res
	}
	return execx.CmdResult{}
}

func TestListInstancesIgnoresAggregateExitWhenValidInstancesExist(t *testing.T) {
	t.Parallel()

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
`

	a := New(fakeRunner{
		results: map[string]execx.CmdResult{
			"glab auth status --all": {
				Stdout:   stdout,
				ExitCode: 1,
			},
		},
	})

	lst, warnings, errs, err := a.ListInstances(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %#v", errs)
	}
	if len(lst.Instances) != 2 {
		t.Fatalf("expected 2 instances, got %d", len(lst.Instances))
	}
}
