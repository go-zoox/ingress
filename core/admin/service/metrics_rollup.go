package service

import (
	"sort"
	"sync"
	"time"

	"github.com/go-zoox/ingress/core/admin/model"
)

const (
	rollupMaxEntries     = 8_000
	rollupRetention      = 30 * time.Minute
	rollupSynthPerBucket = 8
	rollupSynthMaxTotal  = 2_000
)

// RollupWindow is the rollup slice selected for one overview window.
type RollupWindow struct {
	Entries      []AccessEntry
	Source       string
	HasData      bool
	FullCoverage bool
}

// MetricsRollup holds recent access events in-process for live overview metrics.
type MetricsRollup struct {
	mu        sync.RWMutex
	entries   []AccessEntry
	store     *MetricsRollupStore
	pending   map[int64]*minuteDelta
	dbBuckets []model.MetricsMinuteBucket
}

// NewMetricsRollup creates an empty rollup store.
func NewMetricsRollup() *MetricsRollup {
	return &MetricsRollup{
		store:   NewMetricsRollupStore(),
		pending: make(map[int64]*minuteDelta),
	}
}

// Record appends one parsed access event and trims by retention / capacity.
func (r *MetricsRollup) Record(e AccessEntry) {
	if r == nil || e.At.IsZero() {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = append(r.entries, e)
	r.applyPendingLocked(e)
	r.trimLocked(time.Now())
}

// IngestBatch appends many entries (startup backfill). Caller should avoid duplicates.
func (r *MetricsRollup) IngestBatch(entries []AccessEntry) {
	if r == nil || len(entries) == 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	for _, e := range entries {
		if e.At.IsZero() {
			continue
		}
		r.entries = append(r.entries, e)
		r.applyPendingLocked(e)
	}
	r.trimLocked(now)
}

func (r *MetricsRollup) applyPendingLocked(e AccessEntry) {
	if r.pending == nil {
		r.pending = make(map[int64]*minuteDelta)
	}
	key := e.At.Truncate(time.Minute).Unix()
	d := deltaFromEntry(e)
	if cur, ok := r.pending[key]; ok {
		mergeMinuteDelta(cur, d)
		return
	}
	copy := d
	r.pending[key] = &copy
}

// Len returns the number of buffered entries.
func (r *MetricsRollup) Len() int {
	if r == nil {
		return 0
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.entries)
}

// LoadPersisted loads minute buckets from SQLite into memory for long-window queries.
func (r *MetricsRollup) LoadPersisted(retention time.Duration) error {
	if r == nil || r.store == nil {
		return nil
	}
	if retention <= 0 {
		retention = rollupPersistRetention
	}
	since := time.Now().Add(-retention)
	rows, err := r.store.LoadSince(since)
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.dbBuckets = rows
	r.mu.Unlock()
	return nil
}

// FlushPending writes pending minute deltas to SQLite.
func (r *MetricsRollup) FlushPending() error {
	if r == nil || r.store == nil {
		return nil
	}
	r.mu.Lock()
	pending := r.pending
	r.pending = make(map[int64]*minuteDelta)
	r.mu.Unlock()

	for key, d := range pending {
		if d == nil || d.count <= 0 {
			continue
		}
		if err := r.store.ApplyDelta(time.Unix(key, 0), *d); err != nil {
			return err
		}
	}
	return nil
}

// bucketsForRange loads minute aggregates from SQLite for [from, anchor] and merges live pending.
func (r *MetricsRollup) bucketsForRange(from, anchor time.Time) []model.MetricsMinuteBucket {
	if r == nil {
		return nil
	}
	from = from.Truncate(time.Minute)
	anchor = anchor.Truncate(time.Minute)
	byKey := map[int64]model.MetricsMinuteBucket{}

	if r.store != nil {
		if rows, err := r.store.LoadBetween(from, anchor); err == nil {
			for _, b := range rows {
				byKey[b.Minute.Truncate(time.Minute).Unix()] = b
			}
		}
	}

	r.mu.RLock()
	for _, b := range r.dbBuckets {
		if bucketInWindow(b.Minute, from, anchor) {
			key := b.Minute.Truncate(time.Minute).Unix()
			byKey[key] = b
		}
	}
	pending := r.pending
	r.mu.RUnlock()

	for key, d := range pending {
		if d == nil || d.count <= 0 {
			continue
		}
		minute := time.Unix(key, 0).Truncate(time.Minute)
		if !bucketInWindow(minute, from, anchor) {
			continue
		}
		if _, ok := byKey[key]; ok {
			continue
		}
		byKey[key] = model.MetricsMinuteBucket{
			Minute:        minute,
			Count:         d.count,
			S2:            d.s2,
			S3:            d.s3,
			S4:            d.s4,
			S5:            d.s5,
			WAFBlocks:     d.wafBlocks,
			CacheHits:     d.cacheHits,
			DurationSumMs: d.durationSumMs,
			DurationCount: d.durationCount,
			DurationMaxMs: d.durationMaxMs,
			UpstreamSumMs: d.upstreamSumMs,
			UpstreamCount: d.upstreamCount,
		}
	}

	if len(byKey) == 0 {
		return nil
	}
	keys := make([]int64, 0, len(byKey))
	for k := range byKey {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	out := make([]model.MetricsMinuteBucket, 0, len(keys))
	for _, k := range keys {
		out = append(out, byKey[k])
	}
	return out
}

// PurgePersistedOlderThan deletes minute buckets older than retainDays.
func (r *MetricsRollup) PurgePersistedOlderThan(retainDays int) (int64, error) {
	if r == nil || r.store == nil {
		return 0, nil
	}
	return r.store.PurgeOlderThan(retainDays)
}

// EntriesForWindow returns entries when rollup fully covers the window (legacy API).
func (r *MetricsRollup) EntriesForWindow(window string) ([]AccessEntry, string, bool) {
	rw := r.WindowEntries(window, false)
	if !rw.HasData || !rw.FullCoverage {
		return nil, "", false
	}
	return rw.Entries, rw.Source, true
}

// WindowEntries selects rollup data for an overview window.
// requireLive skips DB-only fallback when there are no in-process entries (embedded core cold start).
// Query range spans 2× the window so overview delta (环比) can compare to the previous period.
func (r *MetricsRollup) WindowEntries(window string, requireLive bool) RollupWindow {
	return r.windowEntriesAt(window, time.Now(), requireLive)
}

// LiveEntriesForOverview returns in-process rollup entries for the overview window (no DB synthesis).
func (r *MetricsRollup) LiveEntriesForOverview(window string, anchor time.Time) []AccessEntry {
	if r == nil {
		return nil
	}
	windowDur := parseWindowDuration(window)
	slot := timelineSlotForWindow(windowDur, window)
	alignedStart := timelineWindowStart(anchor, windowDur, slot)
	return r.LiveEntriesInRange(alignedStart, anchor)
}

// LiveEntriesInRange returns in-process rollup entries with full access-log detail in [from, to].
func (r *MetricsRollup) LiveEntriesInRange(from, to time.Time) []AccessEntry {
	if r == nil || to.IsZero() {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]AccessEntry(nil), filterEntriesSince(r.entries, from, to)...)
}

func (r *MetricsRollup) windowEntriesAt(window string, anchor time.Time, requireLive bool) RollupWindow {
	empty := RollupWindow{}
	if r == nil {
		return empty
	}
	windowDur := parseWindowDuration(window)
	slot := timelineSlotForWindow(windowDur, window)
	alignedStart := timelineWindowStart(anchor, windowDur, slot)
	queryStart := alignedStart.Add(-windowDur)

	r.mu.RLock()
	mem := filterEntriesSince(r.entries, queryStart, anchor)
	memCopy := append([]AccessEntry(nil), mem...)
	memEarliest := earliestEntryTime(memCopy)
	var combined []AccessEntry
	hasLive := len(r.entries) > 0
	if hasLive && (memEarliest.IsZero() || memEarliest.After(queryStart)) {
		combined = append(combined, r.synthesizedFromDBRangeLocked(queryStart, memEarliest, anchor)...)
	}
	combined = append(combined, memCopy...)
	if len(combined) == 0 && !requireLive {
		combined = r.synthesizedFromDBRangeLocked(queryStart, time.Time{}, anchor)
	}

	if len(combined) == 0 {
		r.mu.RUnlock()
		return empty
	}

	current := filterEntriesSince(combined, alignedStart, anchor)
	full := rollupCoversWindow(current, windowDur, anchor, window)
	source := "rollup_live"
	if len(memCopy) == 0 {
		source = "rollup_persisted"
	} else if len(combined) > len(memCopy) {
		source = "rollup_hybrid"
	}
	rw := RollupWindow{
		Entries:      combined,
		Source:       source,
		HasData:      len(current) > 0,
		FullCoverage: full,
	}
	r.mu.RUnlock()
	return rw
}

func (r *MetricsRollup) synthesizedFromDBRangeLocked(rangeStart, exclusiveEnd, anchor time.Time) []AccessEntry {
	if len(r.dbBuckets) == 0 || anchor.IsZero() {
		return nil
	}
	startMinute := rangeStart.Truncate(time.Minute)
	endMinute := anchor.Truncate(time.Minute)
	exclusive := exclusiveEnd.Truncate(time.Minute)
	var out []AccessEntry
	for _, b := range r.dbBuckets {
		if b.Minute.Before(startMinute) || b.Minute.After(endMinute) {
			continue
		}
		if !exclusive.IsZero() && !b.Minute.Before(exclusive) {
			continue
		}
		part := entriesFromMinuteBucket(b)
		if len(out)+len(part) > rollupSynthMaxTotal {
			break
		}
		out = append(out, part...)
	}
	return out
}

// DeltaFromStores computes overview delta from persisted minute buckets (+ pending) when entry-level data is incomplete.
func (r *MetricsRollup) DeltaFromStores(window string, anchor time.Time) (OverviewDelta, bool) {
	if r == nil {
		return OverviewDelta{}, false
	}
	windowDur := parseWindowDuration(window)
	slot := timelineSlotForWindow(windowDur, window)
	alignedStart := timelineWindowStart(anchor, windowDur, slot)
	prevStart := alignedStart.Add(-windowDur)

	r.mu.RLock()
	prevAgg := r.aggregateRangeLocked(prevStart, alignedStart)
	curEnd := anchor.Truncate(time.Minute).Add(time.Minute)
	curAgg := r.aggregateRangeLocked(alignedStart, curEnd)
	r.mu.RUnlock()

	if prevAgg.count <= 0 {
		return OverviewDelta{}, false
	}
	prev := snapshotFromMinuteAgg(prevAgg, windowDur)
	cur := snapshotFromMinuteAgg(curAgg, windowDur)
	return computeOverviewDeltaFromSnapshots(cur, prev, windowDur), true
}

func (r *MetricsRollup) aggregateRangeLocked(start, endExclusive time.Time) minuteDelta {
	var total minuteDelta
	if endExclusive.IsZero() {
		return total
	}
	start = start.Truncate(time.Minute)
	endExclusive = endExclusive.Truncate(time.Minute)
	for _, b := range r.dbBuckets {
		if b.Minute.Before(start) || !b.Minute.Before(endExclusive) {
			continue
		}
		mergeMinuteDelta(&total, minuteDelta{
			count: b.Count, s2: b.S2, s3: b.S3, s4: b.S4, s5: b.S5,
			wafBlocks: b.WAFBlocks, cacheHits: b.CacheHits,
			durationSumMs: b.DurationSumMs, durationCount: b.DurationCount, durationMaxMs: b.DurationMaxMs,
			upstreamSumMs: b.UpstreamSumMs, upstreamCount: b.UpstreamCount,
		})
	}
	for key, d := range r.pending {
		if d == nil {
			continue
		}
		minute := time.Unix(key, 0).Truncate(time.Minute)
		if minute.Before(start) || !minute.Before(endExclusive) {
			continue
		}
		mergeMinuteDelta(&total, *d)
	}
	return total
}

func snapshotFromMinuteAgg(d minuteDelta, windowDur time.Duration) entrySnapshot {
	var snap entrySnapshot
	if d.count <= 0 {
		return snap
	}
	snap.total = d.count
	snap.wafBlocks = d.wafBlocks
	errors := d.s4 + d.s5
	snap.errorRate = float64(errors) / float64(d.count) * 100
	snap.cacheHitRate = float64(d.cacheHits) / float64(d.count) * 100
	minutes := windowDur.Minutes()
	if minutes < 1 {
		minutes = 1
	}
	snap.rpm = float64(d.count) / minutes
	if d.durationCount > 0 {
		snap.p95 = d.durationMaxMs
		if snap.p95 <= 0 {
			snap.p95 = d.durationSumMs / float64(d.durationCount)
		}
	}
	return snap
}

func computeOverviewDeltaFromSnapshots(cur, prev entrySnapshot, windowDur time.Duration) OverviewDelta {
	d := OverviewDelta{HasPrevious: true}
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

// rollupCoversWindow matches tailCoversWindow using timeline-aligned window start.
func rollupCoversWindow(entries []AccessEntry, windowDur time.Duration, anchor time.Time, window string) bool {
	if !entriesHaveTimestamps(entries) || len(entries) == 0 {
		return false
	}
	earliest := earliestEntryTime(entries)
	if earliest.IsZero() {
		return false
	}
	slot := timelineSlotForWindow(windowDur, window)
	start := timelineWindowStart(anchor, windowDur, slot)
	return !earliest.After(start)
}

func (r *MetricsRollup) trimLocked(now time.Time) {
	if len(r.entries) == 0 {
		return
	}
	cutoff := now.Add(-rollupRetention)
	start := 0
	for start < len(r.entries) && r.entries[start].At.Before(cutoff) {
		start++
	}
	if start > 0 {
		r.entries = append([]AccessEntry(nil), r.entries[start:]...)
	}
	if len(r.entries) > rollupMaxEntries {
		r.entries = append([]AccessEntry(nil), r.entries[len(r.entries)-rollupMaxEntries:]...)
	}
}
