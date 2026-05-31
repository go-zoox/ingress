package service

import (
	"testing"
	"time"
)

func TestParseMetricsRangeQuery_Absolute(t *testing.T) {
	to := time.Now().Add(-time.Hour).Truncate(time.Minute)
	from := to.Add(-8 * time.Hour)
	q, err := ParseMetricsRangeQuery(from.Format(time.RFC3339), to.Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}
	if !q.From.Equal(from) || !q.To.Equal(to) {
		t.Fatalf("range=%+v want from=%v to=%v", q, from, to)
	}
}

func TestParseMetricsRangeQuery_RequiresBoth(t *testing.T) {
	_, err := ParseMetricsRangeQuery(time.Now().Format(time.RFC3339), "")
	if err == nil {
		t.Fatal("expected error when to missing")
	}
}

func TestMetricsRangeFromWindow(t *testing.T) {
	q := MetricsRangeFromWindow("15m")
	if q.Duration() < 14*time.Minute || q.Duration() > 16*time.Minute {
		t.Fatalf("15m duration=%v", q.Duration())
	}
}

func TestTimelineBucketsForDuration(t *testing.T) {
	if timelineBucketsForDuration(5*time.Minute) != 5 {
		t.Fatal("5m buckets")
	}
	if timelineBucketsForDuration(6*time.Hour) != 12 {
		t.Fatal("6h buckets")
	}
}
