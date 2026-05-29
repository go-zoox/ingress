package service

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseAdminConsoleAccessLog(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for !strings.HasSuffix(root, "ingress") && root != "/" {
		root = filepath.Dir(root)
	}
	path := filepath.Join(root, "examples", "admin-console", "access.log")
	f, err := os.Open(path)
	if err != nil {
		t.Skip(err)
	}
	defer f.Close()

	arrow, parsed := 0, 0
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if !strings.Contains(line, " -> ") {
			continue
		}
		arrow++
		if _, ok := parseAccessLine(line); ok {
			parsed++
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	t.Logf("arrow=%d parsed=%d", arrow, parsed)
	if parsed < 100 {
		t.Fatalf("expected most sample ingress lines to parse, got %d/%d", parsed, arrow)
	}

	logs := &Logs{accessPath: path}
	lines, err := logs.TailIngressAccess(8000)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) < 100 {
		t.Fatalf("TailIngressAccess: got %d lines, want >= 100", len(lines))
	}
	m := NewMetrics(logs, nil)
	out15 := m.Overview("15m")
	out24h := m.Overview("24h")
	if out15.Source != "access_log" {
		t.Fatalf("15m source=%q", out15.Source)
	}
	if !out15.WindowStale {
		t.Fatal("expected window_stale for historical sample log")
	}
	if out15.Total <= 0 {
		t.Fatalf("15m total=%d", out15.Total)
	}
	if out15.Total >= arrow {
		t.Fatalf("15m should filter by window, total=%d file_lines=%d", out15.Total, arrow)
	}
	if out24h.Total < out15.Total {
		t.Fatalf("24h total=%d should be >= 15m total=%d", out24h.Total, out15.Total)
	}
}
