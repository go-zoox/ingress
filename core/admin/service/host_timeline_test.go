package service

import (
	"testing"
	"time"
)

func TestBuildHostTimeline_topNAndOther(t *testing.T) {
	anchor := time.Date(2026, 5, 20, 12, 15, 0, 0, time.Local)
	window := 15 * time.Minute
	slot := window / 3
	windowStart := timelineWindowStart(anchor, window, slot)
	entries := []AccessEntry{
		{At: anchor.Add(-14 * time.Minute), Host: "a.example.com", Status: 200},
		{At: anchor.Add(-13 * time.Minute), Host: "a.example.com", Status: 200},
		{At: anchor.Add(-8 * time.Minute), Host: "b.example.com", Status: 200},
		{At: anchor.Add(-7 * time.Minute), Host: "c.example.com", Status: 200},
		{At: anchor.Add(-2 * time.Minute), Host: "d.example.com", Status: 200},
	}

	series := buildHostTimeline(entries, true, 3, windowStart, anchor, slot, nil, 2)
	if len(series) != 3 {
		t.Fatalf("series=%d want 3 (top2 + other)", len(series))
	}
	if series[0].Name != "a.example.com" || series[0].Total != 2 {
		t.Fatalf("top host=%+v", series[0])
	}
	if series[len(series)-1].Name != "Other" || series[len(series)-1].Total != 2 {
		t.Fatalf("other=%+v", series[len(series)-1])
	}
	if len(series[0].Points) != 3 {
		t.Fatalf("points=%d", len(series[0].Points))
	}
}

func TestAggregateOverview_hostTimeline(t *testing.T) {
	anchor := time.Date(2026, 5, 20, 12, 0, 0, 0, time.Local)
	entries := []AccessEntry{
		{At: anchor.Add(-10 * time.Minute), Host: "api.example.com", Status: 200},
		{At: anchor.Add(-5 * time.Minute), Host: "www.example.com", Status: 200},
		{At: anchor.Add(-4 * time.Minute), Host: "www.example.com", Status: 200},
	}
	out := aggregateOverview(entries, "15m", "access_log")
	if len(out.HostTimeline) == 0 {
		t.Fatal("expected host_timeline")
	}
	if out.HostTimeline[0].Name != "www.example.com" {
		t.Fatalf("top series=%s", out.HostTimeline[0].Name)
	}
}
