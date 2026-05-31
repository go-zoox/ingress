package service

import (
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
// When liveOnly is true, only in-process live entries are used (no DB synthesis).
func (r *MetricsRollup) WindowEntries(window string, liveOnly bool) RollupWindow {
	return r.windowEntriesAt(window, time.Now(), liveOnly)
}

func (r *MetricsRollup) windowEntriesAt(window string, anchor time.Time, liveOnly bool) RollupWindow {
	empty := RollupWindow{}
	if r == nil {
		return empty
	}
	windowDur := parseWindowDuration(window)
	slot := timelineSlotForWindow(windowDur, window)
	alignedStart := timelineWindowStart(anchor, windowDur, slot)

	r.mu.RLock()
	mem := filterEntriesSince(r.entries, alignedStart, anchor)
	memCopy := append([]AccessEntry(nil), mem...)
	r.mu.RUnlock()

	if len(memCopy) > 0 {
		full := rollupCoversWindow(memCopy, windowDur, anchor, window)
		return RollupWindow{
			Entries:      memCopy,
			Source:       "rollup_live",
			HasData:      true,
			FullCoverage: full,
		}
	}
	if liveOnly {
		return empty
	}

	r.mu.RLock()
	dbPart := r.synthesizedFromDBLocked(anchor, windowDur, time.Time{})
	r.mu.RUnlock()

	if len(dbPart) == 0 {
		return empty
	}
	full := rollupCoversWindow(dbPart, windowDur, anchor, window)
	return RollupWindow{
		Entries:      dbPart,
		Source:       "rollup_persisted",
		HasData:      true,
		FullCoverage: full,
	}
}

func (r *MetricsRollup) synthesizedFromDBLocked(anchor time.Time, windowDur time.Duration, memEarliest time.Time) []AccessEntry {
	if len(r.dbBuckets) == 0 {
		return nil
	}
	startMinute := anchor.Add(-windowDur).Truncate(time.Minute)
	anchorMinute := anchor.Truncate(time.Minute)
	var out []AccessEntry
	for _, b := range r.dbBuckets {
		if b.Minute.Before(startMinute) || b.Minute.After(anchorMinute) {
			continue
		}
		if !memEarliest.IsZero() && !b.Minute.Before(memEarliest.Truncate(time.Minute)) {
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
