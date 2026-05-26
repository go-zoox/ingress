package service

import (
	"strings"
	"testing"
)

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
			`203.0.113.44 api.example.com -> api.internal:8080 "GET /api/users HTTP/1.1" 200 12ms cache_hit=0 waf_block=0 upstream_status=200 upstream_response_time=10ms`,
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
		// Zoox file transport prepends timestamp + level before the access log line.
		{
			`2026/05/24 19:51:04 2026/05/24 19:51:04 INFO 127.0.0.1 httpbin.work -> https://httpbin.zcorky.com "GET /ip HTTP/1.1" 200 116ms cache_hit=0 waf_block=0`,
			"httpbin.work", 200, 116,
		},
		// Non-access-log lines (zoox framework output) should be skipped.
		{
			`2026/05/24 19:51:05 2026/05/24 19:51:05 INFO [127.0.0.1:53272][=>] GET /api/v1/metrics/overview`,
			"", 0, 0,
		},
		// Zoox file transport with ANSI-colored log level (e.g. \x1b[34mINFO\x1b[39m).
		{
			"2026/05/24 20:14:50 2026/05/24 20:14:50 \x1b[34mINFO\x1b[39m 127.0.0.1 httpbin.work -> https://httpbin.zcorky.com \"GET /ip HTTP/1.1\" 200 132ms cache_hit=0 waf_block=0",
			"httpbin.work", 200, 132,
		},
		// ANSI-colored zoox framework log should still be skipped.
		{
			"2026/05/24 20:14:53 2026/05/24 20:14:53 \x1b[34mINFO\x1b[39m [127.0.0.1:50447][=>] GET /api/v1/metrics/overview",
			"", 0, 0,
		},
	}
	for _, c := range cases {
		e, ok := parseAccessLine(c.line)
		if c.host == "" {
			if ok {
				t.Fatalf("expected skip but parsed: %q", c.line)
			}
			continue
		}
		if !ok {
			t.Fatalf("parse failed: %q", c.line)
		}
		if e.Host != c.host || e.Status != c.code || e.DurationMs != c.ms {
			t.Fatalf("got %+v want host=%s code=%d ms=%v", e, c.host, c.code, c.ms)
		}
		if strings.Contains(c.line, "upstream_response_time=10ms") {
			if e.ClientIP != "203.0.113.44" || e.Target != "api.internal:8080" || e.UpstreamDurationMs != 10 {
				t.Fatalf("extended fields: %+v", e)
			}
		}
	}
}
