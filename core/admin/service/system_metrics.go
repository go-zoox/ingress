package service

import (
	"runtime"
	"sync"
	"time"
)

const (
	systemSampleInterval = 10 * time.Second
	systemHistoryMax     = 360 // 1 hour at 10s
)

// SystemMetricPoint is one sampled point in the process resource timeline.
type SystemMetricPoint struct {
	Label    string  `json:"label"`
	CPUPct   float64 `json:"cpu_pct"`
	MemoryMB float64 `json:"memory_mb"`
}

// SystemMetricsSnapshot is the current process resource view for the admin overview.
type SystemMetricsSnapshot struct {
	Window     string              `json:"window"`
	CPUPct     float64             `json:"cpu_pct"`
	MemoryMB   float64             `json:"memory_mb"`
	Goroutines int                 `json:"goroutines"`
	NumCPU     int                 `json:"num_cpu"`
	Timeline   []SystemMetricPoint `json:"timeline"`
}

type systemSample struct {
	at       time.Time
	cpuPct   float64
	memoryMB float64
}

// SystemMetrics samples ingress process CPU and memory in the background.
type SystemMetrics struct {
	mu           sync.RWMutex
	samples      []systemSample
	lastCPUTime  float64
	lastSampleAt time.Time
	done         chan bool
}

// NewSystemMetrics creates a process metrics sampler.
func NewSystemMetrics() *SystemMetrics {
	return &SystemMetrics{
		done: make(chan bool, 1),
	}
}

// Start launches the background sampling goroutine.
func (s *SystemMetrics) Start() {
	go s.run()
}

// Stop terminates the background goroutine.
func (s *SystemMetrics) Stop() {
	select {
	case s.done <- true:
	default:
	}
}

func (s *SystemMetrics) run() {
	s.recordSample()

	ticker := time.NewTicker(systemSampleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.recordSample()
		case <-s.done:
			return
		}
	}
}

func (s *SystemMetrics) recordSample() {
	now := time.Now()
	cpuTime := processCPUTimeSeconds()
	memoryMB := processMemoryMB()

	s.mu.Lock()
	defer s.mu.Unlock()

	cpuPct := 0.0
	if !s.lastSampleAt.IsZero() {
		wall := now.Sub(s.lastSampleAt).Seconds()
		if wall > 0 {
			delta := cpuTime - s.lastCPUTime
			if delta < 0 {
				delta = 0
			}
			cpuPct = (delta / wall) / float64(runtime.NumCPU()) * 100
			if cpuPct > 100*float64(runtime.NumCPU()) {
				cpuPct = 100 * float64(runtime.NumCPU())
			}
		}
	}

	s.lastCPUTime = cpuTime
	s.lastSampleAt = now

	s.samples = append(s.samples, systemSample{
		at:       now,
		cpuPct:   round1(cpuPct),
		memoryMB: round1(memoryMB),
	})
	if len(s.samples) > systemHistoryMax {
		s.samples = s.samples[len(s.samples)-systemHistoryMax:]
	}
}

// Snapshot returns aggregated process metrics for the requested window.
func (s *SystemMetrics) Snapshot(window string) SystemMetricsSnapshot {
	dur := parseWindowDuration(window)
	cutoff := time.Now().Add(-dur)

	s.mu.RLock()
	filtered := filterSystemSamples(s.samples, cutoff)
	s.mu.RUnlock()

	out := SystemMetricsSnapshot{
		Window:     normalizeMetricsWindow(window),
		Goroutines: runtime.NumGoroutine(),
		NumCPU:     runtime.NumCPU(),
		Timeline:   buildSystemTimeline(filtered, dur),
	}
	if len(filtered) > 0 {
		last := filtered[len(filtered)-1]
		out.CPUPct = last.cpuPct
		out.MemoryMB = last.memoryMB
	}
	return out
}

func filterSystemSamples(samples []systemSample, cutoff time.Time) []systemSample {
	out := make([]systemSample, 0, len(samples))
	for _, sample := range samples {
		if !sample.at.Before(cutoff) {
			out = append(out, sample)
		}
	}
	return out
}

func buildSystemTimeline(samples []systemSample, dur time.Duration) []SystemMetricPoint {
	if len(samples) == 0 {
		return nil
	}
	buckets := timelineBucketsForWindow(durationToMetricsWindow(dur))
	if buckets <= 0 {
		buckets = 15
	}
	slot := dur / time.Duration(buckets)
	if slot <= 0 {
		slot = time.Minute
	}
	anchor := time.Now()
	if last := samples[len(samples)-1].at; !last.IsZero() {
		anchor = last
	}
	windowStart := alignedTimelineEnd(anchor, slot).Add(-dur)

	type bucketScratch struct {
		cpuSum float64
		cpuN   int
		mem    float64
		memSet bool
	}
	scratches := make([]bucketScratch, buckets)
	out := make([]SystemMetricPoint, buckets)
	for i := range out {
		bucketStart := windowStart.Add(time.Duration(i) * slot)
		out[i].Label = formatTimelineLabel(bucketStart, slot)
	}

	for _, sample := range samples {
		if sample.at.Before(windowStart) || sample.at.After(anchor) {
			continue
		}
		idx := int(sample.at.Sub(windowStart) / slot)
		if idx >= buckets {
			idx = buckets - 1
		}
		if idx < 0 {
			idx = 0
		}
		scratches[idx].cpuSum += sample.cpuPct
		scratches[idx].cpuN++
		scratches[idx].mem = sample.memoryMB
		scratches[idx].memSet = true
	}

	for i := range out {
		sc := scratches[i]
		if sc.cpuN > 0 {
			out[i].CPUPct = round1(sc.cpuSum / float64(sc.cpuN))
		}
		if sc.memSet {
			out[i].MemoryMB = sc.mem
		}
	}
	return out
}

func durationToMetricsWindow(d time.Duration) string {
	switch {
	case d >= 24*time.Hour:
		return "24h"
	case d >= time.Hour:
		return "1h"
	case d <= 5*time.Minute:
		return "5m"
	default:
		return "15m"
	}
}

func normalizeMetricsWindow(window string) string {
	switch window {
	case "24h", "1h", "60m", "5m", "15m":
		if window == "60m" {
			return "1h"
		}
		return window
	default:
		return "15m"
	}
}

func round1(v float64) float64 {
	return float64(int(v*10+0.5)) / 10
}
