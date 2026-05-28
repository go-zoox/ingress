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
	if len(out.Timeline) != 2 {
		t.Fatalf("timeline len=%d want 2", len(out.Timeline))
	}
	if out.CPUPct != 3 || out.MemoryMB != 30 {
		t.Fatalf("latest snapshot=%+v", out)
	}
	if out.Window != "15m" {
		t.Fatalf("window=%q", out.Window)
	}
}

func TestBuildSystemTimeline_downsample(t *testing.T) {
	now := time.Now()
	samples := make([]systemSample, 30)
	for i := range samples {
		samples[i] = systemSample{
			at:       now.Add(time.Duration(i) * time.Minute),
			cpuPct:   float64(i),
			memoryMB: float64(i + 1),
		}
	}
	out := buildSystemTimeline(samples, 15*time.Minute)
	if len(out) != 8 {
		t.Fatalf("len=%d want 8", len(out))
	}
	if out[len(out)-1].CPUPct < 20 {
		t.Fatalf("last point should reflect recent samples: %+v", out[len(out)-1])
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
