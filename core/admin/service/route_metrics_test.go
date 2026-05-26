package service

import (
	"testing"
	"time"

	ingcore "github.com/go-zoox/ingress/core"
	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/ingress/core/service"
)

func TestAggregateAccessEntries_routeFilter(t *testing.T) {
	cfg := &ingcore.Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "api.example.com",
				Backend: rule.Backend{
					Service: service.Service{Name: "api.internal", Port: 8080},
				},
			},
		},
	}
	now := time.Now()
	lines := []string{
		formatTestAccessLine(now.Add(-2*time.Minute), "api.example.com", "GET", "/api/users", 200, 12),
		formatTestAccessLine(now.Add(-1*time.Minute), "api.example.com", "GET", "/api/users", 500, 80),
		formatTestAccessLine(now, "other.example.com", "GET", "/", 200, 5),
	}
	var entries []AccessEntry
	for _, line := range lines {
		filtered := FilterAccessEntriesForRoute(cfg, 0, -1, []string{line})
		entries = append(entries, filtered...)
	}
	m := AggregateAccessEntries(entries, "15m", "access_log")
	if m.Total != 2 {
		t.Fatalf("total=%d want 2", m.Total)
	}
	if len(m.Timeline) == 0 {
		t.Fatal("expected timeline buckets")
	}
	if len(m.Slowest) == 0 {
		t.Fatal("expected slowest samples")
	}
	if len(m.ErrorSamples) == 0 {
		t.Fatal("expected error samples")
	}
	if m.StatusCounts["5xx"] != 1 {
		t.Fatalf("5xx=%d want 1", m.StatusCounts["5xx"])
	}
}

func formatTestAccessLine(at time.Time, host, method, path string, status int, ms int) string {
	return at.Format("2006/01/02 15:04:05") + " 192.0.2.1 " + host +
		" -> upstream:8080 \"" + method + " " + path + " HTTP/1.1\" " +
		itoa(status) + " " + itoa(ms) + "ms cache_hit=0 waf_block=0"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
