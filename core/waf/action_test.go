package waf

import "testing"

func TestNormalizeAction(t *testing.T) {
	t.Parallel()
	cases := []struct {
		action  string
		logOnly bool
		want    string
		err     bool
	}{
		{"", false, ActionBlock, false},
		{"", true, ActionAudit, false},
		{"block", false, ActionBlock, false},
		{"audit", false, ActionAudit, false},
		{"pass", false, ActionPass, false},
		{"log_only", false, ActionAudit, false},
		{"allow", false, ActionPass, false},
		{"BLOCK", false, ActionBlock, false},
		{"nope", false, "", true},
	}
	for _, c := range cases {
		got, err := NormalizeAction(c.action, c.logOnly)
		if c.err {
			if err == nil {
				t.Fatalf("action=%q log_only=%v: want error", c.action, c.logOnly)
			}
			continue
		}
		if err != nil {
			t.Fatalf("action=%q: %v", c.action, err)
		}
		if got != c.want {
			t.Fatalf("action=%q log_only=%v: got %q want %q", c.action, c.logOnly, got, c.want)
		}
	}
}
