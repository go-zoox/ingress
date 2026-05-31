package service

import (
	"encoding/json"
	"testing"
)

func TestComputeOverviewSSEPatch_onlyChangedFields(t *testing.T) {
	prev := OverviewSnapshot{
		Window: "15m",
		System: SystemMetricsSnapshot{
			Window:     "15m",
			CPUPct:     0,
			MemoryMB:   43.8,
			Goroutines: 10,
			NumCPU:     4,
		},
	}
	next := prev
	next.System.CPUPct = 2.7

	patch := computeOverviewSSEPatch(prev, next)
	if patch.isEmpty() {
		t.Fatal("expected patch")
	}
	if patch.Window != "15m" {
		t.Fatalf("window should be included for routing, got %q", patch.Window)
	}
	if patch.Status != nil {
		t.Fatal("status should be omitted")
	}

	var system map[string]any
	if err := json.Unmarshal(patch.System, &system); err != nil {
		t.Fatal(err)
	}
	if len(system) != 1 {
		t.Fatalf("system patch should have one field, got %+v", system)
	}
	if system["cpu_pct"] != 2.7 {
		t.Fatalf("cpu_pct=%v", system["cpu_pct"])
	}
	if _, ok := system["window"]; ok {
		t.Fatal("window should not be in system patch")
	}
}

func TestComputeOverviewSSEPatch_noChanges(t *testing.T) {
	snap := OverviewSnapshot{
		Window:  "15m",
		Metrics: OverviewMetrics{Window: "15m", Total: 10, Source: "access_log"},
		System:  SystemMetricsSnapshot{Window: "15m", CPUPct: 1.0, MemoryMB: 50},
	}
	patch := computeOverviewSSEPatch(snap, snap)
	if !patch.isEmpty() {
		t.Fatalf("expected empty patch, got %+v", patch)
	}
}

func TestComputeOverviewSSEPatch_ignoresLastReload(t *testing.T) {
	prev := OverviewSnapshot{
		Status: OverviewStatus{Version: "1.0.0", LastReload: "2026-05-30T10:00:00+08:00"},
	}
	next := prev
	next.Status.LastReload = "2026-05-30T10:00:05+08:00"

	patch := computeOverviewSSEPatch(prev, next)
	if !patch.isEmpty() {
		t.Fatalf("expected empty patch when only last_reload changed, got status=%s", patch.Status)
	}
}

func TestApplyOverviewSSEPatch_mergesFields(t *testing.T) {
	base := OverviewSnapshot{
		System: SystemMetricsSnapshot{
			Window:   "15m",
			CPUPct:   0,
			MemoryMB: 43.8,
			NumCPU:   4,
		},
	}
	patch := OverviewSSEPatch{
		System: json.RawMessage(`{"cpu_pct":2.7}`),
		Seq:    1,
	}
	out := applyOverviewSSEPatch(base, patch)
	if out.System.CPUPct != 2.7 {
		t.Fatalf("cpu=%v", out.System.CPUPct)
	}
	if out.System.MemoryMB != 43.8 {
		t.Fatalf("memory should be preserved, got %v", out.System.MemoryMB)
	}
	if out.System.Window != "15m" {
		t.Fatalf("window should be preserved")
	}
}

func TestComputeOverviewSSEPatch_metricsScalarOnly(t *testing.T) {
	prev := OverviewSnapshot{
		Metrics: OverviewMetrics{Window: "15m", Total: 100, Source: "access_log", RPM: 1.5},
	}
	next := prev
	next.Metrics.Total = 101

	patch := computeOverviewSSEPatch(prev, next)
	var metrics map[string]any
	if err := json.Unmarshal(patch.Metrics, &metrics); err != nil {
		t.Fatal(err)
	}
	if len(metrics) != 1 {
		t.Fatalf("expected one metrics field, got %+v", metrics)
	}
	if int(metrics["total"].(float64)) != 101 {
		t.Fatalf("total=%v", metrics["total"])
	}
}
