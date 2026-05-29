package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTailIngressAccess_skipsZooxNoise(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "access.log")
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, `2026/05/30 06:00:00 INFO [127.0.0.1:12345][<=] GET /api/v1/metrics/overview 200 +1ms`)
	}
	lines = append(lines,
		`2026/02/21 12:59:02 198.51.100.22 waf-demo.example.com -> httpbin.org:443 "GET / HTTP/1.1" 200 24ms cache_hit=0`,
		`2026/02/21 13:00:48 203.0.113.44 assets.cdn.example.com -> minio.internal:9000 "GET /static/main.js HTTP/1.1" 200 5ms cache_hit=1`,
	)
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	logs := &Logs{accessPath: path}
	got, err := logs.TailIngressAccess(8000)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 ingress lines", len(got))
	}
	if !strings.Contains(got[0], "waf-demo.example.com") {
		t.Fatalf("first line: %q", got[0])
	}
}
