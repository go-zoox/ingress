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

func TestAggregateOverview_timelineRatesAndHostErrors(t *testing.T) {
	anchor := time.Date(2026, 5, 20, 12, 0, 0, 0, time.Local)
	entries := []AccessEntry{
		{At: anchor.Add(-10 * time.Minute), Host: "a.example.com", Status: 200, CacheHit: true},
		{At: anchor.Add(-10 * time.Minute), Host: "a.example.com", Status: 404},
		{At: anchor.Add(-5 * time.Minute), Host: "b.example.com", Status: 500, WAFBlock: true},
		{At: anchor.Add(-5 * time.Minute), Host: "b.example.com", Status: 403, WAFBlock: true},
	}

	out := aggregateOverview(entries, "15m", "access_log")
	if len(out.TopHostsError) == 0 {
		t.Fatal("expected top_hosts_error")
	}
	if out.TopHostsError[0].Name != "b.example.com" || out.TopHostsError[0].ErrorRate != 100 {
		t.Fatalf("top_hosts_error[0]=%+v", out.TopHostsError[0])
	}
	var sawRate bool
	for _, b := range out.Timeline {
		if b.Count > 0 && b.ErrorRate > 0 {
			sawRate = true
		}
		if b.WAFBlocks > 0 {
			if b.ErrorRate <= 0 {
				t.Fatalf("bucket with waf should have error_rate: %+v", b)
			}
		}
	}
	if !sawRate {
		t.Fatalf("timeline missing error_rate: %+v", out.Timeline)
	}
}
