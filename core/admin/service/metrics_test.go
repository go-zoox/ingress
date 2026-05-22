package service

import (
	"testing"
	"time"
)

func TestAggregateOverview_staleLogUsesLatestWindow(t *testing.T) {
	anchor := time.Date(2026, 5, 20, 12, 0, 0, 0, time.Local)
	entries := []AccessEntry{
		{At: anchor.Add(-20 * time.Minute), Host: "api.example.com", Method: "GET", Path: "/", Status: 200, DurationMs: 10},
		{At: anchor.Add(-10 * time.Minute), Host: "api.example.com", Method: "GET", Path: "/v2", Status: 200, DurationMs: 12},
		{At: anchor.Add(-5 * time.Minute), Host: "cdn.example.com", Method: "GET", Path: "/a.js", Status: 404, DurationMs: 3},
		{At: anchor, Host: "api.example.com", Method: "GET", Path: "/health", Status: 500, DurationMs: 100},
	}

	out := aggregateOverview(entries, "15m", "access_log")
	if out.Total != 3 {
		t.Fatalf("total=%d want 3", out.Total)
	}
	totalBuckets := 0
	for _, b := range out.Timeline {
		totalBuckets += b.Count
	}
	if totalBuckets != 3 {
		t.Fatalf("timeline count=%d want 3 buckets=%+v", totalBuckets, out.Timeline)
	}
}

func TestBuildTimeline_respectsAnchor(t *testing.T) {
	anchor := time.Date(2026, 5, 20, 10, 15, 0, 0, time.Local)
	entries := []AccessEntry{
		{At: anchor.Add(-14 * time.Minute), Status: 200},
		{At: anchor.Add(-7 * time.Minute), Status: 404},
		{At: anchor.Add(-1 * time.Minute), Status: 502},
	}
	buckets := buildTimeline(entries, true, 15*time.Minute, 3, anchor)
	if buckets[0].Count+buckets[1].Count+buckets[2].Count != 3 {
		t.Fatalf("unexpected buckets: %+v", buckets)
	}
	if buckets[2].Count != 1 || buckets[2].S5 != 1 {
		t.Fatalf("latest bucket want one 5xx: %+v", buckets[2])
	}
}
