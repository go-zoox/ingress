package core

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-zoox/zoox"
	"github.com/go-zoox/zoox/defaults"
)

type stubAccessMetrics struct {
	events []AccessMetricsEvent
}

func (s *stubAccessMetrics) OnAccessMetrics(event AccessMetricsEvent) {
	s.events = append(s.events, event)
}

func TestLogAccess_emitsMetricsCallback(t *testing.T) {
	app := defaults.Default()
	req := httptest.NewRequest("GET", "http://api.example.com/v1/users", nil)
	req.RemoteAddr = "203.0.113.44:1234"
	req.Header.Set("X-Real-IP", "203.0.113.44")
	rec := httptest.NewRecorder()

	var ctx *zoox.Context
	stub := &stubAccessMetrics{}
	c := &core{accessMetricsCb: stub, app: app}

	app.Use(func(zctx *zoox.Context) {
		ctx = zctx
		c.logAccess(zctx, "api.example.com", "api.internal:8080", "GET", "/v1/users", "HTTP/1.1", 200, 12*time.Millisecond, accessLogMeta{
			CacheHit: true,
		})
	})
	app.ServeHTTP(rec, req)

	if ctx == nil {
		t.Fatal("missing zoox context")
	}
	if len(stub.events) != 1 {
		t.Fatalf("events=%d want 1", len(stub.events))
	}
	ev := stub.events[0]
	if ev.Host != "api.example.com" || ev.Status != 200 || !ev.CacheHit {
		t.Fatalf("unexpected event: %+v", ev)
	}
	if ev.DurationMs != 12 {
		t.Fatalf("duration_ms=%v want 12", ev.DurationMs)
	}
}
