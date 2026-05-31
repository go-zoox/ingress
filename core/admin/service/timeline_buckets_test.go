package service

import (
	"testing"
	"time"
)

func TestBuildTimeline_alignedLabelsStableWithinSlot(t *testing.T) {
	slot := 5 * time.Minute / 8
	anchor1 := time.Date(2026, 5, 31, 23, 27, 10, 0, time.Local)
	anchor2 := anchor1.Add(20 * time.Second)
	entries := []AccessEntry{
		{At: anchor1.Add(-4 * time.Minute), Status: 200},
	}

	b1 := buildTimeline(entries, true, 5*time.Minute, 8, anchor1)
	b2 := buildTimeline(entries, true, 5*time.Minute, 8, anchor2)
	if len(b1) != len(b2) {
		t.Fatalf("bucket count mismatch")
	}
	for i := range b1 {
		if b1[i].Label != b2[i].Label {
			t.Fatalf("label[%d] shifted %q -> %q within slot", i, b1[i].Label, b2[i].Label)
		}
	}
	_ = slot
}

func TestBuildTimeline_uniqueShortWindowLabels(t *testing.T) {
	anchor := time.Date(2026, 5, 31, 23, 30, 0, 0, time.Local)
	buckets := buildTimeline(nil, true, 5*time.Minute, 8, anchor)
	seen := map[string]int{}
	for _, b := range buckets {
		seen[b.Label]++
	}
	for label, n := range seen {
		if n > 1 {
			t.Fatalf("duplicate label %q appears %d times", label, n)
		}
	}
}

func TestBuildTimeline_24hBucketCount(t *testing.T) {
	anchor := time.Date(2026, 5, 31, 12, 0, 0, 0, time.Local)
	entries := []AccessEntry{
		{At: anchor.Add(-2 * time.Hour), Status: 200},
		{At: anchor.Add(-30 * time.Minute), Status: 404},
	}
	buckets := buildTimeline(entries, true, 24*time.Hour, 24, anchor)
	if len(buckets) != 24 {
		t.Fatalf("len=%d want 24", len(buckets))
	}
	var total int
	for _, b := range buckets {
		total += b.Count
	}
	if total != 2 {
		t.Fatalf("total count=%d want 2", total)
	}
}
