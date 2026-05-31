package service

import ingcore "github.com/go-zoox/ingress/core"

// AccessEntryFromCoreEvent maps a core access metrics callback payload to a rollup row.
func AccessEntryFromCoreEvent(ev ingcore.AccessMetricsEvent) AccessEntry {
	return AccessEntry{
		At:                 ev.At,
		ClientIP:           ev.ClientIP,
		RealIP:             ev.RealIP,
		Host:               ev.Host,
		Target:             ev.Target,
		Method:             ev.Method,
		Path:               ev.Path,
		Status:             ev.Status,
		DurationMs:         ev.DurationMs,
		UpstreamStatus:     ev.UpstreamStatus,
		UpstreamDurationMs: ev.UpstreamDurationMs,
		CacheHit:           ev.CacheHit,
		WAFBlock:           ev.WAFBlock,
	}
}
