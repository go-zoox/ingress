package service

import "testing"

func TestLooksLikeAccessLogLine(t *testing.T) {
	cases := []struct {
		line string
		want bool
	}{
		{`203.0.113.44 api.example.com -> api.internal:8080 "GET / HTTP/1.1" 200 12ms`, true},
		{`2026/05/24 19:51:05 INFO [127.0.0.1:53272][=>] GET /api/v1/metrics/overview`, false},
		{`broken line with "GET / HTTP/1.1" but no host`, true},
		{"", false},
	}
	for _, c := range cases {
		if got := looksLikeAccessLogLine(c.line); got != c.want {
			t.Fatalf("line=%q got=%v want=%v", c.line, got, c.want)
		}
	}
}

func TestParseAccessLogLines_mixedBatch(t *testing.T) {
	lines := []string{
		`203.0.113.44 api.example.com -> api.internal:8080 "GET /ok HTTP/1.1" 200 12ms`,
		`2026/05/24 19:51:05 INFO [127.0.0.1:53272][=>] GET /api/v1/metrics/overview`,
		`203.0.113.44 api.example.com -> api.internal:8080 incomplete line cache_hit=0`,
		`not access at all`,
	}
	out := ParseAccessLogLines(lines)
	if len(out.Entries) != 1 {
		t.Fatalf("entries=%d want 1", len(out.Entries))
	}
	if out.IssueSkipped != 1 {
		t.Fatalf("issueSkipped=%d want 1", out.IssueSkipped)
	}
	if len(out.Issues) != 1 {
		t.Fatalf("issues=%d want 1", len(out.Issues))
	}
	if out.Issues[0].Reason != "missing_request" {
		t.Fatalf("reason=%q", out.Issues[0].Reason)
	}
}

func TestFingerprintAccessLogLine_stableAcrossTimestamp(t *testing.T) {
	a := `2026/05/24 19:51:04 203.0.113.44 api.example.com -> api.internal:8080 incomplete line`
	b := `2026/05/24 20:10:00 203.0.113.44 api.example.com -> api.internal:8080 incomplete line`
	if fingerprintAccessLogLine(a) != fingerprintAccessLogLine(b) {
		t.Fatal("expected same fingerprint after timestamp strip")
	}
}
