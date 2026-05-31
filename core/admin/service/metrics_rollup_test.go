package service

import (
	"testing"
	"time"
)

func TestMetricsRollup_EntriesForWindow_covers5m(t *testing.T) {
	r := NewMetricsRollup()
	anchor := time.Now()
	windowDur := 5 * time.Minute
	windowStart := timelineWindowStart(anchor, windowDur, time.Minute)
	r.IngestBatch([]AccessEntry{
		{At: windowStart, Status: 200, Host: "a.example.com"},
		{At: windowStart.Add(2 * time.Minute), Status: 200, Host: "a.example.com"},
		{At: windowStart.Add(4 * time.Minute), Status: 404, Host: "b.example.com"},
	})

	rw := r.windowEntriesAt("5m", anchor, false)
	if !rw.HasData || !rw.FullCoverage {
		t.Fatal("expected rollup to cover 5m window")
	}
	if rw.Source != "rollup_live" {
		t.Fatalf("source=%q want rollup_live", rw.Source)
	}
	if len(rw.Entries) != 3 {
		t.Fatalf("len=%d want 3", len(rw.Entries))
	}
}

func TestMetricsRollup_partialLiveWindow(t *testing.T) {
	r := NewMetricsRollup()
	anchor := time.Now()
	r.Record(AccessEntry{At: anchor.Add(-2 * time.Minute), Status: 200})
	rw := r.WindowEntries("5m", true)
	if !rw.HasData {
		t.Fatal("expected partial live data")
	}
	if rw.FullCoverage {
		t.Fatal("expected partial window coverage")
	}
}

func TestMetricsRollup_closedMinuteStableOnAppend(t *testing.T) {
	r := NewMetricsRollup()
	anchor := time.Now()
	windowDur := 5 * time.Minute
	windowStart := timelineWindowStart(anchor, windowDur, time.Minute)
	r.Record(AccessEntry{At: windowStart, Status: 200})

	minute := windowStart.Add(2 * time.Minute)
	r.Record(AccessEntry{At: minute.Add(10 * time.Second), Status: 200})
	r.Record(AccessEntry{At: minute.Add(20 * time.Second), Status: 200})
	r.Record(AccessEntry{At: minute.Add(30 * time.Second), Status: 200})

	entries1 := r.windowEntriesAt("5m", anchor, false).Entries
	b1 := buildTimeline(entries1, true, 5*time.Minute, 5, anchor)

	r.Record(AccessEntry{At: anchor.Add(-30 * time.Second), Status: 200})

	entries2 := r.windowEntriesAt("5m", anchor, false).Entries
	b2 := buildTimeline(entries2, true, 5*time.Minute, 5, anchor)

	idx := -1
	for i, b := range b1 {
		if b.Label == formatTimelineLabel(minute, time.Minute) {
			idx = i
			break
		}
	}
	if idx < 0 {
		t.Fatal("closed minute bucket not found in timeline")
	}
	if b1[idx].Count != 3 {
		t.Fatalf("before=%d want 3", b1[idx].Count)
	}
	if b2[idx].Count != 3 {
		t.Fatalf("closed minute count changed: %d want 3", b2[idx].Count)
	}
}

func TestMetricsRollup_trimRetention(t *testing.T) {
	r := NewMetricsRollup()
	old := time.Now().Add(-rollupRetention - time.Minute)
	r.Record(AccessEntry{At: old, Status: 200})
	r.Record(AccessEntry{At: time.Now().Add(-time.Minute), Status: 200})
	if r.Len() != 1 {
		t.Fatalf("len=%d want 1 after retention trim", r.Len())
	}
}
