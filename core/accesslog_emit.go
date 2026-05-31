package core

import (
	"net/http"
	"time"

	"github.com/go-zoox/zoox"
)

func (c *core) logAccess(ctx *zoox.Context, host, target, method, path, proto string, status int, dur time.Duration, meta accessLogMeta) {
	if ctx == nil {
		return
	}
	line := formatAccessLog(ctx.Request, host, target, method, path, proto, status, dur, meta)
	ctx.Logger.Infof("%s", line)
	if c.accessMetricsCb != nil {
		c.accessMetricsCb.OnAccessMetrics(buildAccessMetricsEvent(ctx.Request, host, target, method, path, status, dur, meta))
	}
}

func buildAccessMetricsEvent(req *http.Request, host, target, method, path string, status int, dur time.Duration, meta accessLogMeta) AccessMetricsEvent {
	upstreamStatus := meta.UpstreamStatus
	if upstreamStatus == 0 {
		upstreamStatus = status
	}
	upstreamDur := meta.UpstreamResponseTime
	if upstreamDur == 0 {
		upstreamDur = dur
	}
	realIP := ""
	if req != nil {
		realIP = req.Header.Get("X-Real-IP")
	}
	if realIP == "" {
		realIP = accessLogClientIP(req)
	}
	return AccessMetricsEvent{
		At:                 time.Now(),
		ClientIP:           accessLogClientIP(req),
		RealIP:             realIP,
		Host:               host,
		Target:             target,
		Method:             method,
		Path:               path,
		Status:             status,
		DurationMs:         float64(dur.Milliseconds()),
		UpstreamStatus:     upstreamStatus,
		UpstreamDurationMs: float64(upstreamDur.Milliseconds()),
		CacheHit:           meta.CacheHit,
		WAFBlock:           meta.WAFBlock,
	}
}
