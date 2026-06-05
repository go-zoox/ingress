package service

import (
	"sort"
	"strings"
	"time"
)

const hostTimelineTopN = 16

// HostTimelinePoint is one bucket in a per-host traffic series.
type HostTimelinePoint struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

// HostTimelineSeries is request counts over time for one host (or "Other").
type HostTimelineSeries struct {
	Name   string              `json:"name"`
	Total  int                 `json:"total"`
	Points []HostTimelinePoint `json:"points"`
}

// enrichHostTimeline fills HostTimeline from per-request rollup rows when available.
func enrichHostTimeline(out *OverviewMetrics, entries []AccessEntry, windowDur time.Duration) {
	if out == nil {
		return
	}
	entries = entriesWithAccessDetail(entries)
	if len(entries) == 0 {
		return
	}
	bucketCount := len(out.Timeline)
	if bucketCount <= 0 {
		window := strings.TrimSpace(out.Window)
		if window == "" {
			window = "15m"
		}
		bucketCount = timelineBucketsForWindow(window)
	}
	hasTime := entriesHaveTimestamps(entries)

	var (
		anchor      time.Time
		windowStart time.Time
		slot        time.Duration
		filtered    []AccessEntry
	)

	if out.Window == "range" && out.RangeFrom != "" && out.RangeTo != "" {
		from, err1 := time.Parse(time.RFC3339, out.RangeFrom)
		to, err2 := time.Parse(time.RFC3339, out.RangeTo)
		if err1 == nil && err2 == nil && to.After(from) {
			windowDur = to.Sub(from)
			slot = windowDur / time.Duration(bucketCount)
			if slot <= 0 {
				slot = time.Minute
			}
			anchor = to
			windowStart = truncateTime(from, slot)
			filtered = filterEntriesSince(entries, windowStart, anchor)
		}
	}

	if len(filtered) == 0 {
		if windowDur <= 0 {
			window := strings.TrimSpace(out.Window)
			if window == "" || window == "range" {
				window = "15m"
			}
			windowDur = parseWindowDuration(window)
		}
		slot = windowDur / time.Duration(bucketCount)
		if slot <= 0 {
			slot = time.Minute
		}
		var windowStale bool
		filtered, anchor, windowStale = selectOverviewEntries(entries, windowDur, hasTime)
		_ = windowStale
		windowStart = timelineWindowStart(anchor, windowDur, slot)
	}

	out.HostTimeline = buildHostTimeline(filtered, hasTime, bucketCount, windowStart, anchor, slot, out.Timeline, hostTimelineTopN)
}

func buildHostTimeline(
	entries []AccessEntry,
	hasTime bool,
	buckets int,
	windowStart, anchor time.Time,
	slot time.Duration,
	labelSource []TimelineBucket,
	topN int,
) []HostTimelineSeries {
	if buckets <= 0 {
		buckets = 8
	}
	if topN <= 0 {
		topN = hostTimelineTopN
	}
	if len(entries) == 0 {
		return nil
	}

	labels := timelineLabels(buckets, hasTime, windowStart, slot, labelSource)

	hostTotals := map[string]int{}
	for _, e := range entries {
		host := strings.TrimSpace(e.Host)
		if host == "" {
			host = "—"
		}
		hostTotals[host]++
	}
	topHosts := topHostNames(hostTotals, topN)
	topSet := map[string]struct{}{}
	for _, h := range topHosts {
		topSet[h] = struct{}{}
	}

	perHost := map[string][]int{}
	for _, h := range topHosts {
		perHost[h] = make([]int, buckets)
	}
	other := make([]int, buckets)

	if hasTime {
		for _, e := range entries {
			if e.At.Before(windowStart) || e.At.After(anchor) {
				continue
			}
			host := strings.TrimSpace(e.Host)
			if host == "" {
				host = "—"
			}
			idx := timelineIndex(e.At, windowStart, slot, buckets)
			if _, ok := topSet[host]; ok {
				perHost[host][idx]++
			} else {
				other[idx]++
			}
		}
	} else {
		per := len(entries) / buckets
		if per < 1 {
			per = 1
		}
		for i, e := range entries {
			idx := i / per
			if idx >= buckets {
				idx = buckets - 1
			}
			host := strings.TrimSpace(e.Host)
			if host == "" {
				host = "—"
			}
			if _, ok := topSet[host]; ok {
				perHost[host][idx]++
			} else {
				other[idx]++
			}
		}
	}

	out := make([]HostTimelineSeries, 0, len(topHosts)+1)
	for _, name := range topHosts {
		counts := perHost[name]
		total := 0
		for _, n := range counts {
			total += n
		}
		if total <= 0 {
			continue
		}
		out = append(out, HostTimelineSeries{
			Name:   name,
			Total:  total,
			Points: hostTimelinePoints(labels, counts),
		})
	}

	otherTotal := 0
	for _, n := range other {
		otherTotal += n
	}
	if otherTotal > 0 {
		out = append(out, HostTimelineSeries{
			Name:   "Other",
			Total:  otherTotal,
			Points: hostTimelinePoints(labels, other),
		})
	}

	if len(out) == 0 {
		return nil
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name == "Other" {
			return false
		}
		if out[j].Name == "Other" {
			return true
		}
		if out[i].Total != out[j].Total {
			return out[i].Total > out[j].Total
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func timelineLabels(buckets int, hasTime bool, windowStart time.Time, slot time.Duration, labelSource []TimelineBucket) []string {
	labels := make([]string, buckets)
	for i := 0; i < buckets; i++ {
		if i < len(labelSource) && labelSource[i].Label != "" && labelSource[i].Label != "—" {
			labels[i] = labelSource[i].Label
			continue
		}
		if hasTime {
			labels[i] = formatTimelineLabel(windowStart.Add(time.Duration(i)*slot), slot)
		} else {
			labels[i] = formatIndexLabel(i, buckets)
		}
	}
	return labels
}

func topHostNames(totals map[string]int, n int) []string {
	ranked := topN(totals, n)
	out := make([]string, len(ranked))
	for i, row := range ranked {
		out[i] = row.Name
	}
	return out
}

func hostTimelinePoints(labels []string, counts []int) []HostTimelinePoint {
	out := make([]HostTimelinePoint, len(counts))
	for i, n := range counts {
		label := "—"
		if i < len(labels) {
			label = labels[i]
		}
		out[i] = HostTimelinePoint{Label: label, Count: n}
	}
	return out
}
