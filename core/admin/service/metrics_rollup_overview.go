package service

import (
	"time"

	"github.com/go-zoox/ingress/core/admin/model"
)

// AggregateOverviewFromStores builds overview metrics from SQLite minute buckets plus live entries.
// Live in-process entries win for minutes they cover (avoids double count with flushed buckets).
func (r *MetricsRollup) AggregateOverviewFromStores(window, source string, anchor time.Time, liveEntries []AccessEntry) (OverviewMetrics, bool) {
	if r == nil {
		return OverviewMetrics{}, false
	}
	windowDur := parseWindowDuration(window)
	slot := timelineSlotForWindow(windowDur, window)
	bucketCount := timelineBucketsForWindow(window)
	alignedStart := timelineWindowStart(anchor, windowDur, slot)

	r.mu.RLock()
	dbRows := append([]model.MetricsMinuteBucket(nil), r.dbBuckets...)
	pending := r.pending
	r.mu.RUnlock()

	liveInWindow := filterEntriesSince(liveEntries, alignedStart, anchor)
	liveMinutes := map[int64]struct{}{}
	for _, e := range liveInWindow {
		if e.At.IsZero() {
			continue
		}
		liveMinutes[e.At.Truncate(time.Minute).Unix()] = struct{}{}
	}

	hasStore := false
	for _, b := range dbRows {
		if bucketInWindow(b.Minute, alignedStart, anchor) && !minuteKeyInSet(b.Minute, liveMinutes) {
			hasStore = true
			break
		}
	}
	if !hasStore && len(liveInWindow) == 0 {
		if len(pending) == 0 {
			return OverviewMetrics{}, false
		}
		for key := range pending {
			minute := time.Unix(key, 0)
			if bucketInWindow(minute, alignedStart, anchor) && !minuteKeyInSet(minute, liveMinutes) {
				hasStore = true
				break
			}
		}
	}
	if !hasStore && len(liveInWindow) == 0 {
		return OverviewMetrics{}, false
	}

	out := OverviewMetrics{
		Window:       window,
		Source:       source,
		StatusCounts: map[string]int{"2xx": 0, "3xx": 0, "4xx": 0, "5xx": 0},
	}
	timeline := make([]TimelineBucket, bucketCount)
	scratches := make([]timelineBucketScratch, bucketCount)
	windowSlot := windowDur / time.Duration(bucketCount)
	if windowSlot <= 0 {
		windowSlot = time.Minute
	}

	accumulateMinute := func(minute time.Time, d minuteDelta) {
		if d.count <= 0 || !bucketInWindow(minute, alignedStart, anchor) {
			return
		}
		idx := timelineIndex(minute, alignedStart, windowSlot, bucketCount)
		fillTimelineFromMinuteDelta(&timeline[idx], d, &scratches[idx])
		out.Total += d.count
		out.StatusCounts["2xx"] += d.s2
		out.StatusCounts["3xx"] += d.s3
		out.StatusCounts["4xx"] += d.s4
		out.StatusCounts["5xx"] += d.s5
		out.WAFBlocks += d.wafBlocks
		if d.s4+d.s5 > 0 {
			out.ErrorRate += float64(d.s4 + d.s5)
		}
		out.CacheHitRate += float64(d.cacheHits)
	}

	for _, b := range dbRows {
		if minuteKeyInSet(b.Minute, liveMinutes) {
			continue
		}
		accumulateMinute(b.Minute, minuteDeltaFromBucket(b))
	}
	for key, d := range pending {
		if d == nil || minuteKeyInSet(time.Unix(key, 0), liveMinutes) {
			continue
		}
		accumulateMinute(time.Unix(key, 0), *d)
	}

	var histogramDurations []float64
	for _, e := range liveInWindow {
		idx := timelineIndex(e.At, alignedStart, windowSlot, bucketCount)
		fillBucketEntry(&timeline[idx], e, &scratches[idx])
		out.Total++
		cls := statusClass(e.Status)
		out.StatusCounts[cls]++
		if e.Status >= 400 {
			out.ErrorRate++
		}
		histogramDurations = append(histogramDurations, e.DurationMs)
		if e.CacheHit {
			out.CacheHitRate++
		}
		if e.WAFBlock {
			out.WAFBlocks++
		}
	}

	if out.Total == 0 {
		return OverviewMetrics{}, false
	}
	out.ErrorRate = out.ErrorRate / float64(out.Total) * 100
	out.CacheHitRate = out.CacheHitRate / float64(out.Total) * 100
	minutes := windowDur.Minutes()
	if minutes < 1 {
		minutes = 1
	}
	out.RPM = float64(out.Total) / minutes

	finalizeTimelineBuckets(timeline, scratches)
	for i := range timeline {
		if timeline[i].Label == "" {
			bucketStart := alignedStart.Add(time.Duration(i) * windowSlot)
			timeline[i].Label = formatTimelineLabel(bucketStart, windowSlot)
		}
	}
	out.Timeline = timeline
	out.LatencyHistogram = buildLatencyHistogram(histogramDurations)
	out.LatencySLO = buildLatencySLO(liveInWindow)

	prevFiltered := filterEntriesInPreviousWindow(liveEntries, anchor, windowDur, len(liveEntries) > 0 && entriesHaveTimestamps(liveEntries))
	if len(prevFiltered) > 0 {
		out.Delta = computeOverviewDelta(liveInWindow, prevFiltered, windowDur)
	}
	return out, true
}

// AggregateOverviewAbsolute builds overview metrics for a fixed [from, anchor] interval.
func (r *MetricsRollup) AggregateOverviewAbsolute(from, anchor time.Time, source string, liveEntries []AccessEntry) (OverviewMetrics, bool) {
	if r == nil || !anchor.After(from) {
		return OverviewMetrics{}, false
	}
	windowDur := anchor.Sub(from)
	slot := timelineSlotForDuration(windowDur)
	bucketCount := timelineBucketsForDuration(windowDur)
	alignedStart := truncateTime(from, slot)

	dbRows := r.bucketsForRange(from, anchor)

	liveInWindow := filterEntriesSince(liveEntries, alignedStart, anchor)
	liveMinutes := map[int64]struct{}{}
	for _, e := range liveInWindow {
		if e.At.IsZero() {
			continue
		}
		liveMinutes[e.At.Truncate(time.Minute).Unix()] = struct{}{}
	}

	hasStore := false
	for _, b := range dbRows {
		if bucketInWindow(b.Minute, alignedStart, anchor) && !minuteKeyInSet(b.Minute, liveMinutes) {
			hasStore = true
			break
		}
	}
	if !hasStore && len(liveInWindow) == 0 {
		return OverviewMetrics{}, false
	}

	out := OverviewMetrics{
		Window:       "range",
		Source:       source,
		StatusCounts: map[string]int{"2xx": 0, "3xx": 0, "4xx": 0, "5xx": 0},
	}
	timeline := make([]TimelineBucket, bucketCount)
	scratches := make([]timelineBucketScratch, bucketCount)
	windowSlot := windowDur / time.Duration(bucketCount)
	if windowSlot <= 0 {
		windowSlot = time.Minute
	}

	accumulateMinute := func(minute time.Time, d minuteDelta) {
		if d.count <= 0 || !bucketInWindow(minute, alignedStart, anchor) {
			return
		}
		idx := timelineIndex(minute, alignedStart, windowSlot, bucketCount)
		fillTimelineFromMinuteDelta(&timeline[idx], d, &scratches[idx])
		out.Total += d.count
		out.StatusCounts["2xx"] += d.s2
		out.StatusCounts["3xx"] += d.s3
		out.StatusCounts["4xx"] += d.s4
		out.StatusCounts["5xx"] += d.s5
		out.WAFBlocks += d.wafBlocks
		if d.s4+d.s5 > 0 {
			out.ErrorRate += float64(d.s4 + d.s5)
		}
		out.CacheHitRate += float64(d.cacheHits)
	}

	for _, b := range dbRows {
		if minuteKeyInSet(b.Minute, liveMinutes) {
			continue
		}
		accumulateMinute(b.Minute, minuteDeltaFromBucket(b))
	}

	var histogramDurations []float64
	for _, e := range liveInWindow {
		idx := timelineIndex(e.At, alignedStart, windowSlot, bucketCount)
		fillBucketEntry(&timeline[idx], e, &scratches[idx])
		out.Total++
		cls := statusClass(e.Status)
		out.StatusCounts[cls]++
		if e.Status >= 400 {
			out.ErrorRate++
		}
		histogramDurations = append(histogramDurations, e.DurationMs)
		if e.CacheHit {
			out.CacheHitRate++
		}
		if e.WAFBlock {
			out.WAFBlocks++
		}
	}

	if out.Total == 0 {
		return OverviewMetrics{}, false
	}
	out.ErrorRate = out.ErrorRate / float64(out.Total) * 100
	out.CacheHitRate = out.CacheHitRate / float64(out.Total) * 100
	minutes := windowDur.Minutes()
	if minutes < 1 {
		minutes = 1
	}
	out.RPM = float64(out.Total) / minutes

	finalizeTimelineBuckets(timeline, scratches)
	for i := range timeline {
		if timeline[i].Label == "" {
			bucketStart := alignedStart.Add(time.Duration(i) * windowSlot)
			timeline[i].Label = formatTimelineLabel(bucketStart, windowSlot)
		}
	}
	out.Timeline = timeline
	out.LatencyHistogram = buildLatencyHistogram(histogramDurations)
	out.LatencySLO = buildLatencySLO(liveInWindow)
	return out, true
}

func minuteDeltaFromBucket(b model.MetricsMinuteBucket) minuteDelta {
	return minuteDelta{
		count: b.Count, s2: b.S2, s3: b.S3, s4: b.S4, s5: b.S5,
		wafBlocks: b.WAFBlocks, cacheHits: b.CacheHits,
		durationSumMs: b.DurationSumMs, durationCount: b.DurationCount, durationMaxMs: b.DurationMaxMs,
		upstreamSumMs: b.UpstreamSumMs, upstreamCount: b.UpstreamCount,
	}
}

func bucketInWindow(minute, start, anchor time.Time) bool {
	if minute.IsZero() {
		return false
	}
	minute = minute.Truncate(time.Minute)
	return !minute.Before(start.Truncate(time.Minute)) && !minute.After(anchor.Truncate(time.Minute))
}

func minuteKeyInSet(minute time.Time, set map[int64]struct{}) bool {
	if len(set) == 0 {
		return false
	}
	_, ok := set[minute.Truncate(time.Minute).Unix()]
	return ok
}

func timelineIndex(at, windowStart time.Time, slot time.Duration, buckets int) int {
	if slot <= 0 {
		slot = time.Minute
	}
	idx := int(at.Sub(windowStart) / slot)
	if idx >= buckets {
		idx = buckets - 1
	}
	if idx < 0 {
		idx = 0
	}
	return idx
}

func fillTimelineFromMinuteDelta(b *TimelineBucket, d minuteDelta, sc *timelineBucketScratch) {
	if d.count <= 0 {
		return
	}
	b.Count += d.count
	b.S2 += d.s2
	b.S3 += d.s3
	b.S4 += d.s4
	b.S5 += d.s5
	b.WAFBlocks += d.wafBlocks
	sc.errors += d.s4 + d.s5
	sc.cacheHits += d.cacheHits
	if d.durationCount > 0 {
		avg := d.durationSumMs / float64(d.durationCount)
		limit := d.count
		if limit > rollupSynthPerBucket {
			limit = rollupSynthPerBucket
		}
		for i := 0; i < limit; i++ {
			sc.durations = append(sc.durations, avg)
		}
	}
	if d.upstreamCount > 0 {
		avgUp := d.upstreamSumMs / float64(d.upstreamCount)
		limit := d.upstreamCount
		if limit > rollupSynthPerBucket {
			limit = rollupSynthPerBucket
		}
		for i := 0; i < limit; i++ {
			sc.upstreamDurations = append(sc.upstreamDurations, avgUp)
		}
	}
}
