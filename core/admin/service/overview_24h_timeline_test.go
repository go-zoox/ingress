package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOverview24hTimelineHasCounts(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for !strings.HasSuffix(root, "ingress") && root != "/" {
		root = filepath.Dir(root)
	}
	path := filepath.Join(root, "examples", "admin-console", "access.log")
	logs := &Logs{accessPath: path}
	m := NewMetrics(logs, nil)
	out1h := m.Overview("1h")
	out24h := m.Overview("24h")
	sum1h, sum24h := 0, 0
	for _, b := range out1h.Timeline {
		sum1h += b.Count
	}
	for _, b := range out24h.Timeline {
		sum24h += b.Count
	}
	t.Logf("1h total=%d timelineSum=%d buckets=%d", out1h.Total, sum1h, len(out1h.Timeline))
	t.Logf("24h total=%d timelineSum=%d buckets=%d", out24h.Total, sum24h, len(out24h.Timeline))
	if out24h.Total > 0 && sum24h == 0 {
		t.Fatalf("24h has total=%d but timeline empty", out24h.Total)
	}
	if out1h.Total > 0 && sum1h == 0 {
		t.Fatalf("1h has total=%d but timeline empty", out1h.Total)
	}
	if sum24h != out24h.Total {
		t.Fatalf("24h timeline sum=%d != total=%d", sum24h, out24h.Total)
	}
}
