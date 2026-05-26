package service

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

// OverviewMetrics is aggregated access-log stats for the admin dashboard.
type OverviewMetrics struct {
	Window       string           `json:"window"`
	Source       string           `json:"source"`
	Total        int              `json:"total"`
	RPM          float64          `json:"rpm"`
	ErrorRate    float64          `json:"error_rate"`
	P50Ms        float64          `json:"p50_ms"`
	P95Ms        float64          `json:"p95_ms"`
	CacheHitRate float64          `json:"cache_hit_rate"`
	WAFBlocks    int              `json:"waf_blocks"`
	StatusCounts   map[string]int     `json:"status_counts"`
	Timeline       []TimelineBucket   `json:"timeline"`
	TopHosts       []NamedCount       `json:"top_hosts"`
	TopHostsError  []HostErrorStat    `json:"top_hosts_error"`
	Slowest        []SlowRequest      `json:"slowest"`
}

type TimelineBucket struct {
	Label        string  `json:"label"`
	Count        int     `json:"count"`
	S2           int     `json:"2xx"`
	S3           int     `json:"3xx"`
	S4           int     `json:"4xx"`
	S5           int     `json:"5xx"`
	ErrorRate    float64 `json:"error_rate"`
	CacheHitRate float64 `json:"cache_hit_rate"`
	WAFBlocks    int     `json:"waf_blocks"`
}

type NamedCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// HostErrorStat ranks hosts by error share in the overview window.
type HostErrorStat struct {
	Name      string  `json:"name"`
	Count     int     `json:"count"`
	Errors    int     `json:"errors"`
	ErrorRate float64 `json:"error_rate"`
}

type SlowRequest struct {
	Host       string  `json:"host"`
	Method     string  `json:"method"`
	Path       string  `json:"path"`
	Status     int     `json:"status"`
	DurationMs float64 `json:"duration_ms"`
}

// Metrics aggregates access logs for overview charts.
type Metrics struct {
	logs *Logs
}

func NewMetrics(logs *Logs) *Metrics {
	return &Metrics{logs: logs}
}

func (m *Metrics) Overview(window string) OverviewMetrics {
	window = strings.TrimSpace(window)
	if window == "" {
		window = "15m"
	}
	lines, err := m.logs.TailAccess(5000)
	if err != nil {
		return aggregateOverview(nil, window, "error")
	}
	entries := make([]AccessEntry, 0, len(lines))
	for _, line := range lines {
		if e, ok := parseAccessLine(line); ok {
			entries = append(entries, e)
		}
	}

	source := "unconfigured"
	if m.logs != nil && strings.TrimSpace(m.logs.AccessLogPath()) != "" {
		source = "access_log"
		if len(lines) == 0 {
			source = "access_log_empty"
		} else if len(entries) == 0 {
			source = "access_log_parse_fail"
		}
	}

	return aggregateOverview(entries, window, source)
}

func aggregateOverview(entries []AccessEntry, window, source string) OverviewMetrics {
	out := OverviewMetrics{
		Window:       window,
		Source:       source,
		StatusCounts: map[string]int{"2xx": 0, "3xx": 0, "4xx": 0, "5xx": 0},
	}
	if len(entries) == 0 {
		out.Timeline = emptyTimeline(8)
		return out
	}

	windowDur := parseWindowDuration(window)

	hasTime := entriesHaveTimestamps(entries)
	anchor := time.Now()
	filtered := filterEntriesInWindow(entries, anchor, windowDur, hasTime)
	if hasTime && len(filtered) == 0 {
		if latest := latestEntryTime(entries); !latest.IsZero() {
			anchor = latest
			filtered = filterEntriesInWindow(entries, anchor, windowDur, true)
		}
	}
	if !hasTime {
		filtered = entries
	}

	out.Total = len(filtered)
	var durations []float64
	cacheHits := 0
	hostCounts := map[string]int{}

	for _, e := range filtered {
		cls := statusClass(e.Status)
		out.StatusCounts[cls]++
		if e.Status >= 400 {
			out.ErrorRate += 1
		}
		if e.DurationMs > 0 {
			durations = append(durations, e.DurationMs)
		}
		if e.CacheHit {
			cacheHits++
		}
		if e.WAFBlock {
			out.WAFBlocks++
		}
		hostCounts[e.Host]++
	}
	if out.Total > 0 {
		out.ErrorRate = out.ErrorRate / float64(out.Total) * 100
		out.CacheHitRate = float64(cacheHits) / float64(out.Total) * 100
		minutes := windowDur.Minutes()
		if minutes > 0 {
			out.RPM = float64(out.Total) / minutes
		}
	}
	if ps := percentiles(durations, 0.5, 0.95); len(ps) >= 2 {
		out.P50Ms, out.P95Ms = ps[0], ps[1]
	}
	out.TopHosts = topN(hostCounts, 8)
	out.TopHostsError = topHostsByError(filtered, 8)
	out.Slowest = slowest(filtered, 5)
	out.Timeline = buildTimeline(filtered, hasTime, windowDur, 8, anchor)

	return out
}

func parseWindowDuration(window string) time.Duration {
	switch window {
	case "1h", "60m":
		return time.Hour
	case "5m":
		return 5 * time.Minute
	default:
		return 15 * time.Minute
	}
}

func entriesHaveTimestamps(entries []AccessEntry) bool {
	for _, e := range entries {
		if !e.At.IsZero() {
			return true
		}
	}
	return false
}

func latestEntryTime(entries []AccessEntry) time.Time {
	var latest time.Time
	for _, e := range entries {
		if e.At.After(latest) {
			latest = e.At
		}
	}
	return latest
}

func filterEntriesInWindow(entries []AccessEntry, anchor time.Time, window time.Duration, requireTime bool) []AccessEntry {
	if !requireTime {
		return append([]AccessEntry(nil), entries...)
	}
	start := anchor.Add(-window)
	out := make([]AccessEntry, 0, len(entries))
	for _, e := range entries {
		if e.At.IsZero() {
			continue
		}
		if e.At.Before(start) || e.At.After(anchor) {
			continue
		}
		out = append(out, e)
	}
	return out
}

func emptyTimeline(n int) []TimelineBucket {
	b := make([]TimelineBucket, n)
	for i := range b {
		b[i] = TimelineBucket{Label: "—"}
	}
	return b
}

func buildTimeline(entries []AccessEntry, hasTime bool, window time.Duration, buckets int, anchor time.Time) []TimelineBucket {
	if buckets <= 0 {
		buckets = 8
	}
	if anchor.IsZero() {
		anchor = time.Now()
	}
	result := make([]TimelineBucket, buckets)
	for i := range result {
		if hasTime {
			end := anchor.Add(-time.Duration(buckets-1-i) * window / time.Duration(buckets))
			result[i].Label = end.Format("15:04")
		} else {
			result[i].Label = formatIndexLabel(i, buckets)
		}
	}

	if len(entries) == 0 {
		return result
	}

	scratches := make([]timelineBucketScratch, buckets)

	if hasTime {
		start := anchor.Add(-window)
		slot := window / time.Duration(buckets)
		if slot <= 0 {
			slot = time.Minute
		}
		for _, e := range entries {
			if e.At.Before(start) || e.At.After(anchor) {
				continue
			}
			idx := int(e.At.Sub(start) / slot)
			if idx >= buckets {
				idx = buckets - 1
			}
			if idx < 0 {
				idx = 0
			}
			fillBucketEntry(&result[idx], e, &scratches[idx])
		}
		finalizeTimelineBuckets(result, scratches)
		return result
	}

	// No timestamps: spread by line order across buckets.
	per := len(entries) / buckets
	if per < 1 {
		per = 1
	}
	for i, e := range entries {
		idx := i / per
		if idx >= buckets {
			idx = buckets - 1
		}
		fillBucketEntry(&result[idx], e, &scratches[idx])
	}
	finalizeTimelineBuckets(result, scratches)
	return result
}

func formatIndexLabel(i, n int) string {
	if i == n-1 {
		return "最近"
	}
	return "段" + strconv.Itoa(i+1)
}

type timelineBucketScratch struct {
	errors    int
	cacheHits int
}

func fillBucketEntry(b *TimelineBucket, e AccessEntry, scratch *timelineBucketScratch) {
	b.Count++
	switch statusClass(e.Status) {
	case "2xx":
		b.S2++
	case "3xx":
		b.S3++
	case "4xx":
		b.S4++
	case "5xx":
		b.S5++
	}
	if e.Status >= 400 {
		scratch.errors++
	}
	if e.CacheHit {
		scratch.cacheHits++
	}
	if e.WAFBlock {
		b.WAFBlocks++
	}
}

func finalizeTimelineBuckets(buckets []TimelineBucket, scratches []timelineBucketScratch) {
	for i := range buckets {
		if buckets[i].Count <= 0 {
			continue
		}
		sc := scratches[i]
		buckets[i].ErrorRate = float64(sc.errors) / float64(buckets[i].Count) * 100
		buckets[i].CacheHitRate = float64(sc.cacheHits) / float64(buckets[i].Count) * 100
	}
}

func percentiles(vals []float64, ps ...float64) (results []float64) {
	if len(vals) == 0 {
		return make([]float64, len(ps))
	}
	sort.Float64s(vals)
	results = make([]float64, len(ps))
	for i, p := range ps {
		idx := int(float64(len(vals)-1) * p)
		if idx < 0 {
			idx = 0
		}
		if idx >= len(vals) {
			idx = len(vals) - 1
		}
		results[i] = vals[idx]
	}
	return results
}

// ComputePercentiles is the exported version of percentiles for use by handlers.
func ComputePercentiles(vals []float64, ps ...float64) []float64 {
	return percentiles(vals, ps...)
}

func topHostsByError(entries []AccessEntry, n int) []HostErrorStat {
	hostTotal := map[string]int{}
	hostErrors := map[string]int{}
	for _, e := range entries {
		if e.Host == "" {
			continue
		}
		hostTotal[e.Host]++
		if e.Status >= 400 {
			hostErrors[e.Host]++
		}
	}
	all := make([]HostErrorStat, 0, len(hostErrors))
	for host, errs := range hostErrors {
		if errs == 0 {
			continue
		}
		total := hostTotal[host]
		rate := 0.0
		if total > 0 {
			rate = float64(errs) / float64(total) * 100
		}
		all = append(all, HostErrorStat{
			Name: host, Count: total, Errors: errs, ErrorRate: rate,
		})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].ErrorRate == all[j].ErrorRate {
			if all[i].Errors == all[j].Errors {
				return all[i].Name < all[j].Name
			}
			return all[i].Errors > all[j].Errors
		}
		return all[i].ErrorRate > all[j].ErrorRate
	})
	if n > len(all) {
		n = len(all)
	}
	return all[:n]
}

func topN(m map[string]int, n int) []NamedCount {
	type pair struct {
		name  string
		count int
	}
	all := make([]pair, 0, len(m))
	for k, v := range m {
		all = append(all, pair{k, v})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].count == all[j].count {
			return all[i].name < all[j].name
		}
		return all[i].count > all[j].count
	})
	if n > len(all) {
		n = len(all)
	}
	out := make([]NamedCount, n)
	for i := 0; i < n; i++ {
		out[i] = NamedCount{Name: all[i].name, Count: all[i].count}
	}
	return out
}

func slowest(entries []AccessEntry, n int) []SlowRequest {
	cp := append([]AccessEntry(nil), entries...)
	sort.Slice(cp, func(i, j int) bool {
		return cp[i].DurationMs > cp[j].DurationMs
	})
	if n > len(cp) {
		n = len(cp)
	}
	out := make([]SlowRequest, 0, n)
	for i := 0; i < n; i++ {
		e := cp[i]
		if e.DurationMs <= 0 {
			continue
		}
		out = append(out, SlowRequest{
			Host: e.Host, Method: e.Method, Path: e.Path,
			Status: e.Status, DurationMs: e.DurationMs,
		})
	}
	return out
}
