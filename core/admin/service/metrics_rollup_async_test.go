package service

import (
	"testing"
	"time"

	ingcore "github.com/go-zoox/ingress/core"
)

func TestAsyncRollupRecorder_enqueuesOffRequestPath(t *testing.T) {
	r := NewMetricsRollup()
	rec := NewAsyncRollupRecorder(r, 16)
	ev := ingcore.AccessMetricsEvent{
		At:     time.Now(),
		Host:   "a.example.com",
		Method: "GET",
		Path:   "/",
		Status: 200,
	}
	rec.Enqueue(ev)

	deadline := time.Now().Add(2 * time.Second)
	for r.Len() == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if r.Len() != 1 {
		t.Fatalf("len=%d want 1", r.Len())
	}
}

func TestAsyncRollupRecorder_dropsWhenFull(t *testing.T) {
	r := NewMetricsRollup()
	rec := NewAsyncRollupRecorder(r, 1)
	ev := ingcore.AccessMetricsEvent{At: time.Now(), Status: 200}
	rec.Enqueue(ev)
	rec.Enqueue(ev)
	rec.Enqueue(ev)
	if rec.Dropped() == 0 {
		t.Fatal("expected drops when queue is full")
	}
}

func TestOverviewStreamer_ThrottledPushAll_noPanic(t *testing.T) {
	s := NewOverviewStreamer(nil, NewSSEBroker())
	for i := 0; i < 50; i++ {
		s.ThrottledPushAll(100 * time.Millisecond)
	}
	time.Sleep(150 * time.Millisecond)
}
