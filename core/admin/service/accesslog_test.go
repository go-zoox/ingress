package service

import "testing"

func TestParseAccessLine(t *testing.T) {
	cases := []struct {
		line string
		host string
		code int
		ms   float64
	}{
		{
			`203.0.113.44 api.example.com -> api.internal:8080 "GET /api/users HTTP/1.1" 200 12ms cache_hit=0`,
			"api.example.com", 200, 12,
		},
		{
			`2026/05/20 10:18:02 [host: api.example.com, target: api.internal:8080] "POST /api/login HTTP/1.1" 401 8ms`,
			"api.example.com", 401, 8,
		},
		{
			`198.51.100.8 tunnel-a.inlets.example.com -> tunnel-a.tunnel:443 "GET / HTTP/1.1" 502 12003ms`,
			"tunnel-a.inlets.example.com", 502, 12003,
		},
	}
	for _, c := range cases {
		e, ok := parseAccessLine(c.line)
		if !ok {
			t.Fatalf("parse failed: %q", c.line)
		}
		if e.Host != c.host || e.Status != c.code || e.DurationMs != c.ms {
			t.Fatalf("got %+v want host=%s code=%d ms=%v", e, c.host, c.code, c.ms)
		}
	}
}
