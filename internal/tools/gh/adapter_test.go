package gh

import "testing"

func TestNormalizeHostsAndAccountsOrder(t *testing.T) {
	raw := ghAuthStatus{
		Hosts: map[string][]ghAccount{
			"z.example": {
				{Login: "bob", Active: false, State: "success"},
				{Login: "alice", Active: true, State: "success"},
			},
			"github.com": {
				{Login: "oldwinter", Active: true, State: "success"},
			},
		},
	}

	st := normalize(raw)
	if len(st.Hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(st.Hosts))
	}
	if st.Hosts[0].Hostname != "github.com" {
		t.Fatalf("expected github.com first, got %s", st.Hosts[0].Hostname)
	}
	if st.Hosts[1].Hostname != "z.example" {
		t.Fatalf("expected z.example second, got %s", st.Hosts[1].Hostname)
	}

	accts := st.Hosts[1].Accounts
	if len(accts) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(accts))
	}
	if !accts[0].Active || accts[0].Login != "alice" {
		t.Fatalf("expected active alice first, got %#v", accts[0])
	}
	if accts[1].Active || accts[1].Login != "bob" {
		t.Fatalf("expected bob second, got %#v", accts[1])
	}
}
