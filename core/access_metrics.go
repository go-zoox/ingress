package core

import "time"

// AccessMetricsEvent is a structured access log row for live admin metrics rollup.
type AccessMetricsEvent struct {
	At                 time.Time
	ClientIP           string
	RealIP             string
	Host               string
	Target             string
	Method             string
	Path               string
	Status             int
	DurationMs         float64
	UpstreamStatus     int
	UpstreamDurationMs float64
	CacheHit           bool
	WAFBlock           bool
}

// AccessMetricsCallback receives one event per emitted ingress access log line.
type AccessMetricsCallback interface {
	OnAccessMetrics(event AccessMetricsEvent)
}
