package service

import (
	"strings"
	"testing"
	"time"
)

func TestAggregateOverview_historicalFallbackRespectsWindow(t *testing.T) {
	anchor := time.Date(2026, 5, 20, 12, 0, 0, 0, time.Local)
	entries := []AccessEntry{
		{At: anchor.Add(-20 * time.Minute), Host: "api.example.com", Method: "GET", Path: "/", Status: 200, DurationMs: 10},
		{At: anchor.Add(-10 * time.Minute), Host: "api.example.com", Method: "GET", Path: "/v2", Status: 200, DurationMs: 12},
		{At: anchor.Add(-5 * time.Minute), Host: "cdn.example.com", Method: "GET", Path: "/a.js", Status: 404, DurationMs: 3},
		{At: anchor, Host: "api.example.com", Method: "GET", Path: "/health", Status: 500, DurationMs: 100},
	}

	out15 := aggregateOverview(entries, "15m", "access_log")
	if out15.Total != 3 {
		t.Fatalf("15m total=%d want 3", out15.Total)
	}
	if !out15.WindowStale {
		t.Fatal("expected window_stale for 15m")
	}

	out5 := aggregateOverview(entries, "5m", "access_log")
	if out5.Total != 2 {
		t.Fatalf("5m total=%d want 2", out5.Total)
	}
	if !out5.WindowStale {
		t.Fatal("expected window_stale for 5m")
	}
	if out5.RPM <= 0 {
		t.Fatalf("5m rpm=%v", out5.RPM)
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

func TestAggregateOverview_upstreamP95AndBackends(t *testing.T) {
	anchor := time.Date(2026, 5, 20, 12, 0, 0, 0, time.Local)
	entries := []AccessEntry{
		{
			At: anchor.Add(-5 * time.Minute), Host: "api.example.com", Target: "api.internal:8080",
			Status: 200, DurationMs: 20, UpstreamDurationMs: 15,
		},
		{
			At: anchor.Add(-4 * time.Minute), Host: "api.example.com", Target: "api.internal:8080",
			Status: 200, DurationMs: 40, UpstreamDurationMs: 35,
		},
		{
			At: anchor.Add(-3 * time.Minute), Host: "api.example.com", Target: "api.internal:8080",
			Status: 200, DurationMs: 60, UpstreamDurationMs: 55,
		},
		{
			At: anchor.Add(-2 * time.Minute), Host: "cdn.example.com", Target: "minio.internal:9000",
			Status: 200, DurationMs: 5, UpstreamDurationMs: 3, CacheHit: true,
		},
		{
			At: anchor.Add(-1 * time.Minute), Host: "cdn.example.com", Target: "minio.internal:9000",
			Status: 502, DurationMs: 100, UpstreamDurationMs: 95, UpstreamStatus: 502,
		},
	}

	out := aggregateOverview(entries, "15m", "access_log")
	if len(out.TopBackends) != 2 {
		t.Fatalf("top_backends=%d want 2: %+v", len(out.TopBackends), out.TopBackends)
	}
	if out.TopBackends[0].Name != "api.internal:8080" || out.TopBackends[0].Count != 3 {
		t.Fatalf("top backend=%+v", out.TopBackends[0])
	}
	if out.TopBackends[0].UpstreamP95Ms <= 0 {
		t.Fatalf("api backend upstream p95=%v", out.TopBackends[0].UpstreamP95Ms)
	}

	var sawUpstreamP95 bool
	for _, b := range out.Timeline {
		if b.UpstreamP95Ms > 0 {
			sawUpstreamP95 = true
		}
	}
	if !sawUpstreamP95 {
		t.Fatalf("timeline missing upstream_p95_ms: %+v", out.Timeline)
	}
}

func TestParseWindowDuration_24h(t *testing.T) {
	if parseWindowDuration("24h") != 24*time.Hour {
		t.Fatal("24h window")
	}
	if timelineBucketsForWindow("24h") != 24 {
		t.Fatal("24h buckets")
	}
}

func TestParseWindowDuration_6h(t *testing.T) {
	if parseWindowDuration("6h") != 6*time.Hour {
		t.Fatal("6h window")
	}
	if timelineBucketsForWindow("6h") != 12 {
		t.Fatal("6h buckets")
	}
}

func TestComputeOverviewDelta(t *testing.T) {
	anchor := time.Date(2026, 5, 20, 12, 0, 0, 0, time.Local)
	cur := []AccessEntry{
		{At: anchor.Add(-5 * time.Minute), Status: 500, DurationMs: 200},
		{At: anchor.Add(-4 * time.Minute), Status: 200, DurationMs: 50, CacheHit: true},
	}
	prev := []AccessEntry{
		{At: anchor.Add(-20 * time.Minute), Status: 200, DurationMs: 30},
	}
	d := computeOverviewDelta(cur, prev, 15*time.Minute)
	if !d.HasPrevious {
		t.Fatal("expected has_previous")
	}
	if d.TotalPct <= 0 {
		t.Fatalf("total_pct=%v want increase", d.TotalPct)
	}
}

func TestBuildLatencySLO_segments(t *testing.T) {
	entries := []AccessEntry{
		{DurationMs: 0, CacheHit: true},
		{DurationMs: 50},
		{DurationMs: 200},
		{DurationMs: 800},
		{DurationMs: 5000},
	}
	slo := buildLatencySLO(entries)
	if len(slo) != 5 {
		t.Fatalf("len=%d want 5", len(slo))
	}
	if slo[0].Count != 1 || slo[1].Count != 1 || slo[2].Count != 1 || slo[3].Count != 1 || slo[4].Count != 1 {
		t.Fatalf("slo counts=%v", slo)
	}
	if slo[0].Pct != 20 {
		t.Fatalf("cache pct=%v want 20", slo[0].Pct)
	}
}

func TestBuildLatencyHistogram_includesZeroMs(t *testing.T) {
	hist := buildLatencyHistogram([]float64{0, 0, 37, 100})
	var total int
	for _, b := range hist {
		total += b.Count
	}
	if total != 4 {
		t.Fatalf("total bucket count=%d want 4", total)
	}
	if hist[0].Count != 3 {
		t.Fatalf("<50ms=%d want 3 (0ms and 37ms)", hist[0].Count)
	}
	if hist[1].Count != 1 {
		t.Fatalf("50-100=%d want 1", hist[1].Count)
	}
}

func TestTailCoversWindow(t *testing.T) {
	anchor := time.Date(2026, 5, 31, 9, 37, 0, 0, time.Local)
	window := 15 * time.Minute
	entries := []AccessEntry{
		{At: anchor.Add(-10 * time.Minute), Status: 200},
	}
	if tailCoversWindow(entries, window, anchor) {
		t.Fatal("10m span should not cover 15m window")
	}
	entries = append(entries, AccessEntry{At: anchor.Add(-16 * time.Minute), Status: 200})
	if !tailCoversWindow(entries, window, anchor) {
		t.Fatal("16m span should cover 15m window")
	}
}

func TestTailLinesForWindow_defaults(t *testing.T) {
	if overviewTailMaxLines("15m") != 8000 {
		t.Fatalf("15m overview tail cap=%d want 8000", overviewTailMaxLines("15m"))
	}
	if overviewTailMaxLines("5m") > overviewTailMaxLines("15m") {
		t.Fatal("5m tail cap should not exceed 15m cap")
	}
}

func TestTrimParsableLinesFromWindowStart_keepsClosedBucketRows(t *testing.T) {
	windowStart := time.Date(2026, 5, 31, 11, 34, 0, 0, time.Local)
	anchor := windowStart.Add(5 * time.Minute)
	lines := []string{
		`2026/05/31 11:32:10 203.0.113.44 api.example.com -> api.internal:8080 "GET /old HTTP/1.1" 200 5ms cache_hit=0`,
		`2026/05/31 11:34:10 203.0.113.44 api.example.com -> api.internal:8080 "GET /a HTTP/1.1" 200 5ms cache_hit=0`,
		`2026/05/31 11:34:20 203.0.113.44 api.example.com -> api.internal:8080 "GET /b HTTP/1.1" 200 5ms cache_hit=0`,
		`2026/05/31 11:38:50 203.0.113.44 api.example.com -> api.internal:8080 "GET /new HTTP/1.1" 200 5ms cache_hit=0`,
	}
	trimmed := trimParsableLinesFromWindowStart(lines, windowStart, anchor)
	if len(trimmed) != 3 {
		t.Fatalf("trimmed=%d want 3 (drop pre-window only)", len(trimmed))
	}
	if !strings.Contains(trimmed[0], `GET /a`) {
		t.Fatalf("first kept line=%q", trimmed[0])
	}
}

func TestBuildTimeline_closedBucketStableWhenLaterTrafficAppends(t *testing.T) {
	windowStart := time.Date(2026, 5, 31, 11, 34, 0, 0, time.Local)
	anchor := windowStart.Add(5 * time.Minute)
	base := []AccessEntry{
		{At: windowStart.Add(10 * time.Second), Status: 200},
		{At: windowStart.Add(20 * time.Second), Status: 200},
		{At: windowStart.Add(30 * time.Second), Status: 404},
	}
	withLater := append(append([]AccessEntry(nil), base...),
		AccessEntry{At: windowStart.Add(4*time.Minute + 30*time.Second), Status: 200},
		AccessEntry{At: windowStart.Add(4*time.Minute + 40*time.Second), Status: 200},
	)

	b1 := buildTimeline(base, true, 5*time.Minute, 5, anchor)
	b2 := buildTimeline(withLater, true, 5*time.Minute, 5, anchor)
	if b1[0].Count != 3 || b2[0].Count != 3 {
		t.Fatalf("closed bucket counts changed: before=%d after=%d", b1[0].Count, b2[0].Count)
	}
	if b2[4].Count <= b1[4].Count {
		t.Fatalf("latest bucket should grow: before=%d after=%d", b1[4].Count, b2[4].Count)
	}
}

func TestTailIncludesWindowStart(t *testing.T) {
	windowStart := time.Date(2026, 5, 31, 11, 34, 0, 0, time.Local)
	inside := []string{
		`2026/05/31 11:33:59 203.0.113.44 api.example.com -> api.internal:8080 "GET /a HTTP/1.1" 200 5ms cache_hit=0`,
		`2026/05/31 11:34:10 203.0.113.44 api.example.com -> api.internal:8080 "GET /b HTTP/1.1" 200 5ms cache_hit=0`,
	}
	outside := []string{
		`2026/05/31 11:35:01 203.0.113.44 api.example.com -> api.internal:8080 "GET /a HTTP/1.1" 200 5ms cache_hit=0`,
	}
	if !tailIncludesWindowStart(inside, windowStart) {
		t.Fatal("expected window start covered when earliest is before window start")
	}
	if tailIncludesWindowStart(outside, windowStart) {
		t.Fatal("expected window start missing when earliest is after start")
	}
}
