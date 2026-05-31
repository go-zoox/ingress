package service

import (
	"testing"
	"time"
)

func TestMetricsRangeForSubscriber_PresetWindow(t *testing.T) {
	sub := &Subscriber{Params: map[string]string{"window": "15m"}}
	q := metricsRangeForSubscriber(sub)
	if q.Duration() < 14*time.Minute || q.Duration() > 16*time.Minute {
		t.Fatalf("expected ~15m duration, got %v", q.Duration())
	}
}

func TestMetricsRangeForSubscriber_AbsoluteRollingEnd(t *testing.T) {
	from := time.Now().Add(-2 * time.Hour).Format(time.RFC3339)
	to := time.Now().Add(-30 * time.Second).Format(time.RFC3339)
	sub := &Subscriber{Params: map[string]string{"from": from, "to": to}}
	q := metricsRangeForSubscriber(sub)
	if time.Since(q.To) > 5*time.Second {
		t.Fatalf("expected to clamped near now, got %v", q.To)
	}
}

func TestMetricsRangeForSubscriber_AbsoluteHistorical(t *testing.T) {
	from := time.Now().Add(-48 * time.Hour).Format(time.RFC3339)
	to := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	sub := &Subscriber{Params: map[string]string{"from": from, "to": to}}
	q := metricsRangeForSubscriber(sub)
	if !q.To.Before(time.Now().Add(-23 * time.Hour)) {
		t.Fatalf("expected historical to unchanged, got %v", q.To)
	}
}

func TestSubscriberRangeCacheKey(t *testing.T) {
	sub := &Subscriber{Params: map[string]string{"window": "5m"}}
	if subscriberRangeCacheKey(sub) != "window:5m" {
		t.Fatalf("unexpected key %q", subscriberRangeCacheKey(sub))
	}
	sub.Params["from"] = "2026-01-01T00:00:00Z"
	sub.Params["to"] = "2026-01-01T01:00:00Z"
	if subscriberRangeCacheKey(sub) != "range:2026-01-01T00:00:00Z|2026-01-01T01:00:00Z" {
		t.Fatalf("unexpected key %q", subscriberRangeCacheKey(sub))
	}
}
