package service

import (
	"testing"
	"time"
)

func TestOverview_withRealLogData(t *testing.T) {
	// Simulate what happens when we parse the real log file lines
	// Lines that contain real access log entries
	lines := []string{
		`2026/05/24 20:10:00 2026/05/24 20:10:00 INFO 127.0.0.1 httpbin.work -> https://httpbin.zcorky.com "GET /ip HTTP/1.1" 200 125ms cache_hit=0 waf_block=0 real_ip=127.0.0.1 referer=- ua=curl/7.86.0 xff=- tls_protocol=- tls_cipher=- upstream_status=200 upstream_response_length=55 upstream_response_time=125ms`,
		`2026/05/24 20:10:00 2026/05/24 20:10:00 INFO 127.0.0.1 httpbin.work -> https://httpbin.zcorky.com "GET /ip HTTP/1.1" 200 37ms cache_hit=0 waf_block=0 real_ip=127.0.0.1`,
		`2026/05/24 20:10:01 2026/05/24 20:10:01 INFO 127.0.0.1 httpbin.work -> https://httpbin.zcorky.com "GET /ip HTTP/1.1" 200 38ms cache_hit=0 waf_block=0`,
		// Zoox framework logs that should be skipped
		`2026/05/24 20:11:33 INFO [127.0.0.1:64965][=>] GET /api/v1/metrics/overview`,
		`2026/05/24 20:11:33 INFO [127.0.0.1:64965][<=] GET /api/v1/metrics/overview 200 +6ms`,
		// Startup logs
		`2026/05/24 20:11:31 INFO Server started at http://127.0.0.1:8080`,
	}

	entries := make([]AccessEntry, 0)
	for _, line := range lines {
		if e, ok := parseAccessLine(line); ok {
			t.Logf("Parsed: host=%s method=%s path=%s status=%d at=%v", e.Host, e.Method, e.Path, e.Status, e.At)
			entries = append(entries, e)
		}
	}

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	// Now test aggregation
	result := aggregateOverview(entries, "15m", "access_log")
	t.Logf("Total: %d, RPM: %.1f, ErrorRate: %.1f", result.Total, result.RPM, result.ErrorRate)
	if result.Total != 3 {
		t.Fatalf("expected total=3, got %d", result.Total)
	}

	// Check timestamps are parsed
	hasTime := entriesHaveTimestamps(entries)
	t.Logf("hasTime: %v", hasTime)
	if !hasTime {
		t.Fatal("expected entries to have timestamps")
	}

	// Check time-based filtering: entries are ~3 minutes ago, window is 15m
	// anchor is time.Now(), entries should be within window
	anchor := time.Now()
	windowDur := parseWindowDuration("15m")
	filtered := filterEntriesInWindow(entries, anchor, windowDur, true)
	t.Logf("filtered count: %d (anchor=%v)", len(filtered), anchor)
	if len(filtered) == 0 {
		t.Fatal("expected filtered entries within 15m window, got 0")
	}
}

func TestOverview_ansiColoredLog(t *testing.T) {
	// Exact format from real log file with ANSI escapes
	lines := []string{
		"2026/05/24 20:14:50 2026/05/24 20:14:50 \x1b[34mINFO\x1b[39m 127.0.0.1 httpbin.work -> https://httpbin.zcorky.com \"GET /ip HTTP/1.1\" 200 132ms cache_hit=0 waf_block=0 real_ip=127.0.0.1 referer=- ua=curl/7.86.0",
		"2026/05/24 20:14:50 2026/05/24 20:14:50 \x1b[34mINFO\x1b[39m 127.0.0.1 httpbin.work -> https://httpbin.zcorky.com \"GET /ip HTTP/1.1\" 200 37ms cache_hit=0",
	}

	entries := make([]AccessEntry, 0)
	for _, line := range lines {
		if e, ok := parseAccessLine(line); ok {
			t.Logf("Parsed: host=%s method=%s status=%d at=%v", e.Host, e.Method, e.Status, e.At)
			entries = append(entries, e)
		} else {
			t.Logf("FAILED to parse: %q", line)
		}
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	result := aggregateOverview(entries, "15m", "access_log")
	if result.Total != 2 {
		t.Fatalf("expected total=2, got %d", result.Total)
	}
}
