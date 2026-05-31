package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMigrateAccessLogToBuckets(t *testing.T) {
	setupMetricsRollupDB(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "access.log")
	content := "" +
		"2026/05/31 10:00:01 1.2.3.4 api.example.com -> up:8080 \"GET /a HTTP/1.1\" 200 12ms cache_hit=1\n" +
		"2026/05/31 10:00:02 1.2.3.4 api.example.com -> up:8080 \"GET /b HTTP/1.1\" 404 20ms cache_hit=0\n" +
		"2026/05/31 10:01:01 1.2.3.4 api.example.com -> up:8080 \"GET /c HTTP/1.1\" 200 5ms cache_hit=1\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	since := time.Date(2026, 5, 31, 10, 0, 0, 0, time.Local)
	res, err := MigrateAccessLogToBuckets(path, MigrateAccessLogOptions{Since: since})
	if err != nil {
		t.Fatal(err)
	}
	if res.LinesParsed != 3 {
		t.Fatalf("parsed=%d want 3", res.LinesParsed)
	}
	if res.MinutesInserted != 2 {
		t.Fatalf("inserted=%d want 2", res.MinutesInserted)
	}

	store := NewMetricsRollupStore()
	rows, err := store.LoadSince(since)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows=%d want 2", len(rows))
	}
	if rows[0].Count+rows[1].Count != 3 {
		t.Fatalf("counts=%d+%d", rows[0].Count, rows[1].Count)
	}

	res2, err := MigrateAccessLogToBuckets(path, MigrateAccessLogOptions{Since: since})
	if err != nil {
		t.Fatal(err)
	}
	if res2.MinutesInserted != 0 {
		t.Fatalf("second insert=%d want 0", res2.MinutesInserted)
	}
	if res2.MinutesReplaced != 2 {
		t.Fatalf("replaced=%d want 2", res2.MinutesReplaced)
	}
	rows2, err := store.LoadSince(since)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows2) != 2 || rows2[0].Count+rows2[1].Count != 3 {
		t.Fatalf("counts changed after re-migrate: %+v", rows2)
	}
}
