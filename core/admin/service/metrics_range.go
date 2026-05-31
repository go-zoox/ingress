package service

import (
	"fmt"
	"strings"
	"time"
)

const maxMetricsRangeDuration = 30 * 24 * time.Hour

// MetricsRangeQuery is an absolute time interval [From, To].
type MetricsRangeQuery struct {
	From time.Time
	To   time.Time
}

// ParseMetricsRangeQuery parses RFC3339 ?from=&to=.
func ParseMetricsRangeQuery(fromStr, toStr string) (MetricsRangeQuery, error) {
	fromStr = strings.TrimSpace(fromStr)
	toStr = strings.TrimSpace(toStr)
	if fromStr == "" || toStr == "" {
		return MetricsRangeQuery{}, fmt.Errorf("from and to are required (RFC3339)")
	}
	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		return MetricsRangeQuery{}, fmt.Errorf("invalid from: %w", err)
	}
	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		return MetricsRangeQuery{}, fmt.Errorf("invalid to: %w", err)
	}
	return NormalizeMetricsRange(from, to)
}

// NormalizeMetricsRange clamps and validates an interval.
func NormalizeMetricsRange(from, to time.Time) (MetricsRangeQuery, error) {
	if !to.After(from) {
		return MetricsRangeQuery{}, fmt.Errorf("to must be after from")
	}
	if to.Sub(from) > maxMetricsRangeDuration {
		return MetricsRangeQuery{}, fmt.Errorf("range exceeds maximum of 30 days")
	}
	now := time.Now()
	if to.After(now) {
		to = now
	}
	if !to.After(from) {
		return MetricsRangeQuery{}, fmt.Errorf("to must be after from")
	}
	return MetricsRangeQuery{From: from, To: to}, nil
}

// MetricsRangeFromWindow converts a legacy rolling window to [now-duration, now].
func MetricsRangeFromWindow(window string) MetricsRangeQuery {
	to := time.Now()
	from := to.Add(-parseWindowDuration(normalizeMetricsWindow(window)))
	return MetricsRangeQuery{From: from, To: to}
}

func (q MetricsRangeQuery) Duration() time.Duration {
	if q.To.Before(q.From) {
		return 0
	}
	return q.To.Sub(q.From)
}

func WindowLabelForDuration(d time.Duration) string {
	if d <= 0 {
		return "15m"
	}
	minutes := int(d.Round(time.Minute) / time.Minute)
	switch {
	case minutes <= 5:
		return "5m"
	case minutes <= 15:
		return "15m"
	case minutes <= 60:
		return "1h"
	case minutes <= 6*60:
		return "6h"
	default:
		return "24h"
	}
}

func timelineBucketsForDuration(d time.Duration) int {
	if d <= 0 {
		return 5
	}
	minutes := int(d.Minutes())
	switch {
	case minutes <= 15:
		if minutes < 5 {
			return 5
		}
		return minutes
	case minutes <= 60:
		return 12
	case minutes <= 6*60:
		return 12
	case minutes <= 24*60:
		return 24
	default:
		return 24
	}
}

func timelineSlotForDuration(d time.Duration) time.Duration {
	buckets := timelineBucketsForDuration(d)
	if buckets <= 0 {
		return time.Minute
	}
	slot := d / time.Duration(buckets)
	if slot <= 0 {
		return time.Minute
	}
	return slot
}
