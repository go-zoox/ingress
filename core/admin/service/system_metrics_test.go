package service

import (
	"testing"
	"time"
)

func TestSystemMetricsSnapshot_windowFilter(t *testing.T) {
	svc := NewSystemMetrics()
	now := time.Now()
	svc.mu.Lock()
	svc.samples = []systemSample{
		{at: now.Add(-20 * time.Minute), cpuPct: 1, memoryMB: 10},
		{at: now.Add(-10 * time.Minute), cpuPct: 2, memoryMB: 20},
		{at: now.Add(-2 * time.Minute), cpuPct: 3, memoryMB: 30},
	}
	svc.mu.Unlock()

	out := svc.Snapshot("15m")
	if len(out.Timeline) != 8 {
		t.Fatalf("timeline len=%d want 8 buckets for 15m", len(out.Timeline))
	}
	if out.CPUPct != 3 || out.MemoryMB != 30 {
		t.Fatalf("latest snapshot=%+v", out)
	}
	if out.Window != "15m" {
		t.Fatalf("window=%q", out.Window)
	}
}

func TestBuildSystemTimeline_bucketed(t *testing.T) {
	now := time.Now()
	samples := make([]systemSample, 20)
	for i := range samples {
		samples[i] = systemSample{
			at:       now.Add(-time.Duration(19-i) * time.Minute),
			cpuPct:   float64(i + 1),
			memoryMB: float64(i + 10),
		}
	}
	out := buildSystemTimeline(samples, 15*time.Minute)
	if len(out) != 8 {
		t.Fatalf("len=%d want 8", len(out))
	}
	if out[len(out)-1].CPUPct <= 0 {
		t.Fatalf("last bucket should include recent samples: %+v", out[len(out)-1])
	}
}

func TestNormalizeMetricsWindow(t *testing.T) {
	if normalizeMetricsWindow("60m") != "1h" {
		t.Fatal("expected 1h")
	}
	if normalizeMetricsWindow("") != "15m" {
		t.Fatal("expected default 15m")
	}
}
