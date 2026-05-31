package service

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

// OverviewMetrics is aggregated access-log stats for the admin dashboard.
type OverviewMetrics struct {
	Window           string           `json:"window"`
	Source           string           `json:"source"`
	Total            int              `json:"total"`
	RPM              float64          `json:"rpm"`
	ErrorRate        float64          `json:"error_rate"`
	P50Ms            float64          `json:"p50_ms"`
	P95Ms            float64          `json:"p95_ms"`
	CacheHitRate     float64          `json:"cache_hit_rate"`
	WAFBlocks        int              `json:"waf_blocks"`
	StatusCounts     map[string]int   `json:"status_counts"`
	Timeline         []TimelineBucket `json:"timeline"`
	TopHosts         []NamedCount     `json:"top_hosts"`
	TopHostsError    []HostErrorStat  `json:"top_hosts_error"`
	HostTraffic      []HostTrafficStat `json:"host_traffic,omitempty"`
	TopPaths         []NamedCount     `json:"top_paths"`
	TopBackends      []BackendStat    `json:"top_backends"`
	Slowest          []SlowRequest    `json:"slowest"`
	ErrorSamples     []SlowRequest    `json:"error_samples,omitempty"`
	LatencyHistogram []LatencyBucket  `json:"latency_histogram"`
	Delta            OverviewDelta    `json:"delta"`
	ParseSkipped     int              `json:"parse_skipped"`
	ParseIssueOpen   int              `json:"parse_issue_open"`
	ParseableInTail  int              `json:"parseable_in_tail"`
	WindowStale      bool             `json:"window_stale"`
}

type TimelineBucket struct {
	Label         string  `json:"label"`
	Count         int     `json:"count"`
	S2            int     `json:"2xx"`
	S3            int     `json:"3xx"`
	S4            int     `json:"4xx"`
	S5            int     `json:"5xx"`
	ErrorRate     float64 `json:"error_rate"`
	CacheHitRate  float64 `json:"cache_hit_rate"`
	WAFBlocks     int     `json:"waf_blocks"`
	P50Ms         float64 `json:"p50_ms"`
	P95Ms         float64 `json:"p95_ms"`
	UpstreamP95Ms float64 `json:"upstream_p95_ms"`
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

// HostTrafficStat is per-host page views and unique visitors in the metrics window.
type HostTrafficStat struct {
	Name string `json:"name"`
	PV   int    `json:"pv"`
	UV   int    `json:"uv"`
}

// BackendStat aggregates traffic and upstream latency for one backend target.
type BackendStat struct {
	Name             string  `json:"name"`
	Count            int     `json:"count"`
	RPM              float64 `json:"rpm"`
	UpstreamP95Ms    float64 `json:"upstream_p95_ms"`
	UpstreamErrorPct float64 `json:"upstream_error_pct"`
}

type SlowRequest struct {
	Host       string  `json:"host"`
	Method     string  `json:"method"`
	Path       string  `json:"path"`
	Status     int     `json:"status"`
	DurationMs float64 `json:"duration_ms"`
}

// LatencyBucket is a histogram bucket for request duration_ms.
type LatencyBucket struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

// OverviewDelta compares the current window to the immediately previous window of equal length.
type OverviewDelta struct {
	TotalPct       float64 `json:"total_pct"`
	RpmPct         float64 `json:"rpm_pct"`
	ErrorRateDelta float64 `json:"error_rate_delta"`
	CacheHitDelta  float64 `json:"cache_hit_delta"`
	WafBlocksDelta int     `json:"waf_blocks_delta"`
	P95DeltaMs     float64 `json:"p95_delta_ms"`
	HasPrevious    bool    `json:"has_previous"`
}

// Metrics aggregates access logs for overview charts.
type Metrics struct {
	logs        *Logs
	parseIssues *ParseIssues
}

func NewMetrics(logs *Logs, parseIssues *ParseIssues) *Metrics {
	return &Metrics{logs: logs, parseIssues: parseIssues}
}

func (m *Metrics) Overview(window string) OverviewMetrics {
	window = strings.TrimSpace(window)
	if window == "" {
		window = "15m"
	}
	lines, err := m.tailIngressAccessForWindow(window)
	if err != nil {
		return aggregateOverview(nil, window, "error")
	}
	parseResult := ParseAccessLogLines(lines)
	entries := parseResult.Entries
	parseSkipped := parseResult.IssueSkipped
	if m.parseIssues != nil && len(parseResult.Issues) > 0 {
		_ = m.parseIssues.RecordCandidates(parseResult.Issues)
	}

	source := m.overviewSource(lines, entries, parseSkipped)

	out := aggregateOverview(entries, window, source)
	out.ParseSkipped = parseSkipped
	out.ParseableInTail = len(entries)
	if m.parseIssues != nil {
		if n, err := m.parseIssues.OpenCount(); err == nil {
			out.ParseIssueOpen = int(n)
		}
	}
	return out
}

func (m *Metrics) tailIngressAccessForWindow(window string) ([]string, error) {
	if m.logs == nil {
		return nil, nil
	}
	windowDur := parseWindowDuration(window)
	limit := tailLinesForWindow(window)
	maxLimit := maxTailLinesForWindow(window)
	for {
		lines, err := m.logs.TailIngressAccess(limit)
		if err != nil {
			return nil, err
		}
		parseResult := ParseAccessLogLines(lines)
		if tailCoversWindow(parseResult.Entries, windowDur, time.Now()) || limit >= maxLimit {
			return lines, nil
		}
		next := limit * 2
		if next > maxLimit {
			next = maxLimit
		}
		if next <= limit {
			return lines, nil
		}
		limit = next
	}
}

func (m *Metrics) overviewSource(lines []string, entries []AccessEntry, parseSkipped int) string {
	if m.logs == nil {
		return "unconfigured"
	}
	path := strings.TrimSpace(m.logs.AccessLogPath())
	if path == "" {
		return "unconfigured"
	}
	size, err := LogFileSize(path)
	if err != nil || size == 0 {
		return "access_log_empty"
	}
	if len(entries) > 0 || len(lines) > 0 || parseSkipped > 0 {
		return "access_log"
	}
	return "access_log_empty"
}

func aggregateOverview(entries []AccessEntry, window, source string) OverviewMetrics {
	out := OverviewMetrics{
		Window:       window,
		Source:       source,
		StatusCounts: map[string]int{"2xx": 0, "3xx": 0, "4xx": 0, "5xx": 0},
	}
	bucketCount := timelineBucketsForWindow(window)
	if len(entries) == 0 {
		out.Timeline = emptyTimeline(bucketCount)
		return out
	}

	windowDur := parseWindowDuration(window)

	hasTime := entriesHaveTimestamps(entries)
	filtered, anchor, windowStale := selectOverviewEntries(entries, windowDur, hasTime)

	out.Total = len(filtered)
	out.WindowStale = windowStale
	var durations []float64
	cacheHits := 0
	hostCounts := map[string]int{}
	pathCounts := map[string]int{}

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
		pathCounts[accessPathKey(e.Path)]++
	}
	if out.Total > 0 {
		out.ErrorRate = out.ErrorRate / float64(out.Total) * 100
		out.CacheHitRate = float64(cacheHits) / float64(out.Total) * 100
		minutes := windowDur.Minutes()
		if minutes < 1 {
			minutes = 1
		}
		out.RPM = float64(out.Total) / minutes
	}
	if ps := percentiles(durations, 0.5, 0.95); len(ps) >= 2 {
		out.P50Ms, out.P95Ms = ps[0], ps[1]
	}
	out.TopHosts = topN(hostCounts, 8)
	out.TopHostsError = topHostsByError(filtered, 8)
	out.HostTraffic = hostTrafficStats(filtered, 12)
	out.TopPaths = topN(pathCounts, 6)
	out.TopBackends = topBackends(filtered, windowDur, 8)
	out.Slowest = slowest(filtered, 5)
	out.ErrorSamples = errorSamples(filtered, 5)
	out.LatencyHistogram = buildLatencyHistogram(durations)
	out.Timeline = buildTimeline(filtered, hasTime, windowDur, bucketCount, anchor)

	prevFiltered := filterEntriesInPreviousWindow(entries, anchor, windowDur, hasTime)
	out.Delta = computeOverviewDelta(filtered, prevFiltered, windowDur)

	return out
}

func accessPathKey(path string) string {
	pathKey := path
	if pathKey == "" {
		pathKey = "/"
	}
	if i := strings.Index(pathKey, "?"); i >= 0 {
		pathKey = pathKey[:i]
	}
	return pathKey
}

// ScopeHostPathCounts lists every host and path seen in entries for the given window.
func ScopeHostPathCounts(entries []AccessEntry, window string) (hosts, paths []NamedCount) {
	filtered := entriesInMetricsWindow(entries, window)
	if len(filtered) == 0 {
		return nil, nil
	}
	hostCounts := map[string]int{}
	pathCounts := map[string]int{}
	for _, e := range filtered {
		hostCounts[e.Host]++
		pathCounts[accessPathKey(e.Path)]++
	}
	return topN(hostCounts, len(hostCounts)), topN(pathCounts, len(pathCounts))
}

// entriesInMetricsWindow applies the same time-window filter as aggregateOverview.
func entriesInMetricsWindow(entries []AccessEntry, window string) []AccessEntry {
	if len(entries) == 0 {
		return nil
	}
	filtered, _, _ := selectOverviewEntries(entries, parseWindowDuration(window), entriesHaveTimestamps(entries))
	return filtered
}

// selectOverviewEntries returns entries for the overview window. When nothing falls
// inside the live window, entries are filtered from the latest log timestamp backward
// (historical mode) instead of returning the full file.
func selectOverviewEntries(entries []AccessEntry, windowDur time.Duration, hasTime bool) (filtered []AccessEntry, anchor time.Time, windowStale bool) {
	anchor = time.Now()
	if !hasTime {
		return entries, anchor, false
	}
	filtered = filterEntriesInWindow(entries, anchor, windowDur, true)
	if len(filtered) > 0 {
		return filtered, anchor, false
	}
	if latest := latestEntryTime(entries); !latest.IsZero() {
		anchor = latest
	}
	filtered = filterEntriesInWindow(entries, anchor, windowDur, true)
	return filtered, anchor, true
}

func hostTrafficStats(entries []AccessEntry, limit int) []HostTrafficStat {
	if len(entries) == 0 {
		return nil
	}
	cap := limit
	if cap == 0 {
		cap = 12
	}
	pv := map[string]int{}
	uvSets := map[string]map[string]struct{}{}
	for _, e := range entries {
		host := strings.TrimSpace(e.Host)
		if host == "" {
			continue
		}
		pv[host]++
		ip := visitorIP(e)
		if ip == "" {
			continue
		}
		if uvSets[host] == nil {
			uvSets[host] = map[string]struct{}{}
		}
		uvSets[host][ip] = struct{}{}
	}
	out := make([]HostTrafficStat, 0, len(pv))
	for host, count := range pv {
		uv := 0
		if set := uvSets[host]; set != nil {
			uv = len(set)
		}
		out = append(out, HostTrafficStat{Name: host, PV: count, UV: uv})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].PV != out[j].PV {
			return out[i].PV > out[j].PV
		}
		return out[i].Name < out[j].Name
	})
	if cap > 0 && len(out) > cap {
		out = out[:cap]
	}
	return out
}

func parseWindowDuration(window string) time.Duration {
	switch window {
	case "24h":
		return 24 * time.Hour
	case "1h", "60m":
		return time.Hour
	case "5m":
		return 5 * time.Minute
	default:
		return 15 * time.Minute
	}
}

// TailLinesForWindow returns the initial tail budget for a metrics window (may expand adaptively).
func TailLinesForWindow(window string) int {
	return tailLinesForWindow(window)
}

func tailLinesForWindow(window string) int {
	switch window {
	case "24h":
		return 50000
	case "1h", "60m":
		return 25000
	case "5m":
		return 10000
	default:
		return 20000
	}
}

func maxTailLinesForWindow(window string) int {
	switch window {
	case "24h":
		return maxLogScanLines
	case "1h", "60m":
		return 150000
	case "5m":
		return 40000
	default:
		return 80000
	}
}

func tailCoversWindow(entries []AccessEntry, windowDur time.Duration, anchor time.Time) bool {
	if !entriesHaveTimestamps(entries) || len(entries) == 0 {
		return true
	}
	earliest := earliestEntryTime(entries)
	if earliest.IsZero() {
		return true
	}
	return !earliest.After(anchor.Add(-windowDur))
}

func timelineBucketsForWindow(window string) int {
	switch window {
	case "24h":
		return 24
	case "1h", "60m":
		return 12
	case "5m":
		return 5
	case "15m":
		return 15
	default:
		return 15
	}
}

func filterEntriesInPreviousWindow(entries []AccessEntry, anchor time.Time, window time.Duration, requireTime bool) []AccessEntry {
	if !requireTime || anchor.IsZero() {
		return nil
	}
	start := anchor.Add(-2 * window)
	end := anchor.Add(-window)
	out := make([]AccessEntry, 0, len(entries)/4)
	for _, e := range entries {
		if e.At.IsZero() {
			continue
		}
		if e.At.Before(start) || e.At.After(end) {
			continue
		}
		out = append(out, e)
	}
	return out
}

func computeOverviewDelta(current, previous []AccessEntry, windowDur time.Duration) OverviewDelta {
	d := OverviewDelta{}
	if len(previous) == 0 {
		return d
	}
	d.HasPrevious = true

	cur := snapshotEntries(current, windowDur)
	prev := snapshotEntries(previous, windowDur)

	if prev.total > 0 {
		d.TotalPct = pctChange(float64(cur.total), float64(prev.total))
		d.RpmPct = pctChange(cur.rpm, prev.rpm)
	}
	d.ErrorRateDelta = cur.errorRate - prev.errorRate
	d.CacheHitDelta = cur.cacheHitRate - prev.cacheHitRate
	d.WafBlocksDelta = cur.wafBlocks - prev.wafBlocks
	d.P95DeltaMs = cur.p95 - prev.p95
	return d
}

type entrySnapshot struct {
	total        int
	rpm          float64
	errorRate    float64
	cacheHitRate float64
	wafBlocks    int
	p95          float64
}

func snapshotEntries(entries []AccessEntry, windowDur time.Duration) entrySnapshot {
	var snap entrySnapshot
	if len(entries) == 0 {
		return snap
	}
	snap.total = len(entries)
	var durations []float64
	errors := 0
	cacheHits := 0
	for _, e := range entries {
		if e.Status >= 400 {
			errors++
		}
		if e.CacheHit {
			cacheHits++
		}
		if e.WAFBlock {
			snap.wafBlocks++
		}
		if e.DurationMs > 0 {
			durations = append(durations, e.DurationMs)
		}
	}
	snap.errorRate = float64(errors) / float64(snap.total) * 100
	snap.cacheHitRate = float64(cacheHits) / float64(snap.total) * 100
	if minutes := windowDur.Minutes(); minutes > 0 {
		snap.rpm = float64(snap.total) / minutes
	}
	if ps := percentiles(durations, 0.95); len(ps) > 0 {
		snap.p95 = ps[0]
	}
	return snap
}

func pctChange(cur, prev float64) float64 {
	if prev == 0 {
		if cur == 0 {
			return 0
		}
		return 100
	}
	return (cur - prev) / prev * 100
}

var latencyHistBounds = []struct {
	label string
	maxMs float64
}{
	{"<50ms", 50},
	{"50-100", 100},
	{"100-250", 250},
	{"250-500", 500},
	{"500ms-1s", 1000},
	{"1-3s", 3000},
	{">3s", 1e18},
}

func buildLatencyHistogram(durations []float64) []LatencyBucket {
	out := make([]LatencyBucket, len(latencyHistBounds))
	for i, b := range latencyHistBounds {
		out[i].Label = b.label
	}
	for _, ms := range durations {
		if ms <= 0 {
			continue
		}
		for i, b := range latencyHistBounds {
			if ms <= b.maxMs {
				out[i].Count++
				break
			}
		}
	}
	return out
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

func earliestEntryTime(entries []AccessEntry) time.Time {
	var earliest time.Time
	for _, e := range entries {
		if e.At.IsZero() {
			continue
		}
		if earliest.IsZero() || e.At.Before(earliest) {
			earliest = e.At
		}
	}
	return earliest
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
	slot := window / time.Duration(buckets)
	if slot <= 0 {
		slot = time.Minute
	}
	windowStart := timelineWindowStart(anchor, window, slot)

	result := make([]TimelineBucket, buckets)
	for i := range result {
		if hasTime {
			bucketStart := windowStart.Add(time.Duration(i) * slot)
			result[i].Label = formatTimelineLabel(bucketStart, slot)
		} else {
			result[i].Label = formatIndexLabel(i, buckets)
		}
	}

	if len(entries) == 0 {
		return result
	}

	scratches := make([]timelineBucketScratch, buckets)

	if hasTime {
		for _, e := range entries {
			if e.At.Before(windowStart) || e.At.After(anchor) {
				continue
			}
			idx := int(e.At.Sub(windowStart) / slot)
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
	errors            int
	cacheHits         int
	durations         []float64
	upstreamDurations []float64
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
	if e.DurationMs > 0 {
		scratch.durations = append(scratch.durations, e.DurationMs)
	}
	if !e.CacheHit {
		up := effectiveUpstreamDurationMs(e)
		if up > 0 {
			scratch.upstreamDurations = append(scratch.upstreamDurations, up)
		}
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
		if ps := percentiles(sc.durations, 0.5, 0.95); len(ps) >= 2 {
			buckets[i].P50Ms = ps[0]
			buckets[i].P95Ms = ps[1]
		}
		if ps := percentiles(sc.upstreamDurations, 0.95); len(ps) >= 1 {
			buckets[i].UpstreamP95Ms = ps[0]
		}
	}
}

func effectiveUpstreamDurationMs(e AccessEntry) float64 {
	if e.UpstreamDurationMs > 0 {
		return e.UpstreamDurationMs
	}
	return 0
}

func effectiveUpstreamStatus(e AccessEntry) int {
	if e.UpstreamStatus > 0 {
		return e.UpstreamStatus
	}
	return e.Status
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

type backendScratch struct {
	count          int
	proxyRequests  int
	upstreamErrors int
	upstreamDurs   []float64
}

func topBackends(entries []AccessEntry, windowDur time.Duration, n int) []BackendStat {
	scratches := map[string]*backendScratch{}
	for _, e := range entries {
		target := strings.TrimSpace(e.Target)
		if target == "" || target == "handler" {
			continue
		}
		s := scratches[target]
		if s == nil {
			s = &backendScratch{}
			scratches[target] = s
		}
		s.count++
		if e.CacheHit {
			continue
		}
		s.proxyRequests++
		if up := effectiveUpstreamDurationMs(e); up > 0 {
			s.upstreamDurs = append(s.upstreamDurs, up)
		}
		if effectiveUpstreamStatus(e) >= 500 {
			s.upstreamErrors++
		}
	}
	all := make([]BackendStat, 0, len(scratches))
	minutes := windowDur.Minutes()
	for name, s := range scratches {
		st := BackendStat{
			Name:  name,
			Count: s.count,
		}
		if minutes > 0 {
			st.RPM = float64(s.count) / minutes
		}
		if s.proxyRequests > 0 {
			st.UpstreamErrorPct = float64(s.upstreamErrors) / float64(s.proxyRequests) * 100
		}
		if ps := percentiles(s.upstreamDurs, 0.95); len(ps) >= 1 {
			st.UpstreamP95Ms = ps[0]
		}
		all = append(all, st)
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Count == all[j].Count {
			return all[i].Name < all[j].Name
		}
		return all[i].Count > all[j].Count
	})
	if n > len(all) {
		n = len(all)
	}
	return all[:n]
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

// AggregateAccessEntries builds dashboard-style metrics from parsed access log lines.
func AggregateAccessEntries(entries []AccessEntry, window, source string) OverviewMetrics {
	return aggregateOverview(entries, window, source)
}

func errorSamples(entries []AccessEntry, n int) []SlowRequest {
	cp := make([]AccessEntry, 0, len(entries))
	for _, e := range entries {
		if e.Status >= 400 {
			cp = append(cp, e)
		}
	}
	if len(cp) == 0 {
		return nil
	}
	sort.Slice(cp, func(i, j int) bool {
		if !cp[i].At.IsZero() && !cp[j].At.IsZero() {
			return cp[i].At.After(cp[j].At)
		}
		return cp[i].DurationMs > cp[j].DurationMs
	})
	if n > len(cp) {
		n = len(cp)
	}
	out := make([]SlowRequest, 0, n)
	for i := 0; i < n; i++ {
		e := cp[i]
		out = append(out, SlowRequest{
			Host: e.Host, Method: e.Method, Path: e.Path,
			Status: e.Status, DurationMs: e.DurationMs,
		})
	}
	return out
}
