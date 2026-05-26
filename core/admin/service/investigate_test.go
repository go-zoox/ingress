package service

import "testing"

func TestFilterAccessEntries(t *testing.T) {
	lines := []string{
		`203.0.113.1 api.example.com -> upstream:8080 "GET /api/users HTTP/1.1" 200 10ms cache_hit=0`,
		`203.0.113.1 api.example.com -> upstream:8080 "GET /api/other HTTP/1.1" 404 5ms cache_hit=0`,
		`203.0.113.1 other.example.com -> upstream:8080 "GET /api/users HTTP/1.1" 200 8ms cache_hit=0`,
	}
	entries := FilterAccessEntries(lines, "api.example.com", "/api/users", "", 10)
	if len(entries) != 1 {
		t.Fatalf("got %d entries want 1", len(entries))
	}
	if entries[0].Path != "/api/users" || entries[0].Status != 200 {
		t.Fatalf("got %+v", entries[0])
	}
}

func TestStatsFromEntries(t *testing.T) {
	stats := StatsFromEntries([]AccessEntry{
		{Status: 200, DurationMs: 10, CacheHit: true},
		{Status: 500, DurationMs: 100, CacheHit: false},
	})
	if stats.Count != 2 || stats.ErrorRate != 50 || stats.CacheHitRate != 50 {
		t.Fatalf("got %+v", stats)
	}
}
