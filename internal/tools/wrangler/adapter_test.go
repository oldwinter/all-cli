package wrangler

import "testing"

func TestWhoamiParsesAccountIDs(t *testing.T) {
	text := `
Getting User settings...
👋 You are logged in with an OAuth Token, associated with the email (redacted).
┌──────────────┬──────────────────────────────────┐
│ Account Name │ Account ID                       │
├──────────────┼──────────────────────────────────┤
│ (redacted)   │ 3ba1294bcdfb7a6f8c113ebc120411df │
│ (redacted)   │ 2371c3163e63aba96bd280648d9ffffc │
│ (redacted)   │ 0ed12f90b68226a08b1a38f0010e99f2 │
└──────────────┴──────────────────────────────────┘
`
	ids := uniqueSorted(accountIDRe.FindAllString(text, -1))
	if len(ids) != 3 {
		t.Fatalf("expected 3 ids, got %d: %#v", len(ids), ids)
	}
}
