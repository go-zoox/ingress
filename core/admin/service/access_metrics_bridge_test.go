package service

import (
	"testing"
	"time"

	ingcore "github.com/go-zoox/ingress/core"
)

func TestAccessEntryFromCoreEvent(t *testing.T) {
	at := time.Now()
	ev := ingcore.AccessMetricsEvent{
		At:                 at,
		Host:               "api.example.com",
		Status:             403,
		DurationMs:         5,
		WAFBlock:           true,
		UpstreamDurationMs: 4,
	}
	e := AccessEntryFromCoreEvent(ev)
	if e.Host != ev.Host || e.Status != 403 || !e.WAFBlock || e.DurationMs != 5 {
		t.Fatalf("entry=%+v", e)
	}
}
