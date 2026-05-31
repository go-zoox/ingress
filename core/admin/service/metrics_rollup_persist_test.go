package service

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/go-zoox/gormx"
	"github.com/go-zoox/ingress/core/admin/model"
)

func setupMetricsRollupDB(t *testing.T) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "rollup.db")
	dsn := "file:" + dbPath + "?cache=shared&_fk=1"
	if err := gormx.LoadDB("sqlite", dsn); err != nil {
		t.Fatalf("load db: %v", err)
	}
	if err := gormx.GetDB().AutoMigrate(model.MigrateModels()...); err != nil {
		t.Fatalf("migrate: %v", err)
	}
}

func TestMetricsRollupStore_ApplyDeltaAndLoad(t *testing.T) {
	setupMetricsRollupDB(t)
	store := NewMetricsRollupStore()
	minute := time.Now().UTC().Truncate(time.Minute)

	if err := store.ApplyDelta(minute, minuteDelta{count: 2, s2: 2, cacheHits: 1}); err != nil {
		t.Fatal(err)
	}
	if err := store.ApplyDelta(minute, minuteDelta{count: 1, s4: 1}); err != nil {
		t.Fatal(err)
	}

	rows, err := store.LoadSince(minute.Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%d want 1", len(rows))
	}
	if rows[0].Count != 3 || rows[0].S2 != 2 || rows[0].S4 != 1 || rows[0].CacheHits != 1 {
		t.Fatalf("row=%+v", rows[0])
	}
}

func TestMetricsRollup_requireLiveSkipsDBOnly(t *testing.T) {
	setupMetricsRollupDB(t)
	r := NewMetricsRollup()
	anchor := time.Now()
	if err := r.store.ApplyDelta(anchor.Add(-5*time.Minute).Truncate(time.Minute), minuteDelta{count: 5, s2: 5}); err != nil {
		t.Fatal(err)
	}
	if err := r.LoadPersisted(rollupPersistRetention); err != nil {
		t.Fatal(err)
	}
	rw := r.WindowEntries("15m", true)
	if rw.HasData {
		t.Fatal("requireLive should ignore DB-only buckets without in-process entries")
	}
}

func TestMetricsRollup_deltaRangeIncludesPreviousWindow(t *testing.T) {
	r := NewMetricsRollup()
	anchor := time.Now()
	windowDur := 15 * time.Minute
	slot := timelineSlotForWindow(windowDur, "15m")
	curStart := timelineWindowStart(anchor, windowDur, slot)
	prevStart := curStart.Add(-windowDur)

	r.Record(AccessEntry{At: prevStart.Add(2 * time.Minute), Status: 200, DurationMs: 30})
	r.Record(AccessEntry{At: prevStart.Add(10 * time.Minute), Status: 200, DurationMs: 40})
	r.Record(AccessEntry{At: curStart.Add(3 * time.Minute), Status: 500, DurationMs: 200})
	r.Record(AccessEntry{At: curStart.Add(8 * time.Minute), Status: 200, DurationMs: 50, CacheHit: true})

	rw := r.windowEntriesAt("15m", anchor, true)
	if len(rw.Entries) < 4 {
		t.Fatalf("expected 2x window entries for delta, got %d", len(rw.Entries))
	}
	out := aggregateOverview(rw.Entries, "15m", rw.Source)
	if !out.Delta.HasPrevious {
		t.Fatal("expected has_previous delta from 2x rollup window")
	}
}

func TestMetricsRollup_persistedOnlyWindow(t *testing.T) {
	setupMetricsRollupDB(t)
	r := NewMetricsRollup()
	anchor := time.Now()

	for i := 15; i >= 1; i-- {
		minute := anchor.Add(-time.Duration(i) * time.Minute).Truncate(time.Minute)
		if err := r.store.ApplyDelta(minute, minuteDelta{count: 1, s2: 1}); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.LoadPersisted(rollupPersistRetention); err != nil {
		t.Fatal(err)
	}

	got, source, ok := r.EntriesForWindow("15m")
	if !ok {
		t.Fatal("expected persisted coverage")
	}
	if source != "rollup_persisted" {
		t.Fatalf("source=%q want rollup_persisted", source)
	}
	if len(got) != 15 {
		t.Fatalf("len=%d want 15", len(got))
	}
}

func TestMetrics_Overview_liveHookNeverTails(t *testing.T) {
	m := NewMetrics(nil, nil)
	m.SetLiveHook(true)
	out := m.Overview("5m")
	if out.Source != "rollup_live" {
		t.Fatalf("source=%q want rollup_live", out.Source)
	}
}
