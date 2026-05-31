package service

import (
	"strconv"
	"strings"

	"github.com/go-zoox/ingress/core/admin/model"
	ingcore "github.com/go-zoox/ingress/core"
	"github.com/go-zoox/ingress/core/rule"
)

// RouteMetricsBuilder builds route detail metrics from access logs.
type RouteMetricsBuilder struct {
	ingress *Ingress
	health  *HealthCheckService
	audit   *Audit
}

// NewRouteMetricsBuilder creates a route metrics builder.
func NewRouteMetricsBuilder(ingress *Ingress, health *HealthCheckService, audit *Audit) *RouteMetricsBuilder {
	return &RouteMetricsBuilder{ingress: ingress, health: health, audit: audit}
}

// Build returns analytics for one route and optional scope filters.
func (b *RouteMetricsBuilder) Build(
	cfg *ingcore.Config,
	ruleIndex, pathIndex int,
	window string,
	rangeQ MetricsRangeQuery,
	scopeHost, scopePath, pathMatch string,
) RouteAnalytics {
	window = strings.TrimSpace(window)
	if window == "" {
		window = WindowLabelForDuration(rangeQ.Duration())
	}

	lines := []string(nil)
	if b != nil && b.ingress != nil && b.ingress.Logs() != nil {
		var err error
		lines, err = b.ingress.Logs().TailAccess(TailLinesForWindow(window))
		if err != nil {
			return BuildRouteAnalytics(
				cfg, ruleIndex, pathIndex,
				window, rangeQ, nil,
				b.health,
				nil,
				false, 0, 0,
				scopeHost, scopePath, pathMatch,
			)
		}
	}

	cacheEnabled := false
	var cacheTTL, cacheMaxBodyKB int64
	if cfg != nil && ruleIndex >= 0 && ruleIndex < len(cfg.Rules) {
		r := &cfg.Rules[ruleIndex]
		var backend rule.Backend
		if pathIndex >= 0 && pathIndex < len(r.Paths) {
			backend = r.Paths[pathIndex].Backend
		} else {
			backend = r.Backend
		}
		if backend.Cache.Enabled {
			cacheEnabled = true
			cacheTTL = backend.Cache.TTL
			cacheMaxBodyKB = backend.Cache.MaxBodyBytes / 1024
		}
	}

	var wafEvents []model.WAFEvent
	if b != nil && b.audit != nil {
		rows, _ := b.audit.ListWAFEvents(WAFAuditFilter{Limit: 500})
		wafEvents = rows
		if cfg != nil && len(rows) > 0 {
			wafEvents = FilterWAFEventsForRoute(cfg, ruleIndex, pathIndex, rows)
		}
	}

	health := (*HealthCheckService)(nil)
	if b != nil {
		health = b.health
	}
	return BuildRouteAnalytics(
		cfg, ruleIndex, pathIndex,
		window, rangeQ, lines,
		health,
		wafEvents,
		cacheEnabled, cacheTTL, cacheMaxBodyKB,
		scopeHost, scopePath, pathMatch,
	)
}

// RouteAnalyticsToMap serializes route analytics for REST/SSE payloads.
func RouteAnalyticsToMap(a RouteAnalytics) map[string]any {
	m := a.OverviewMetrics
	out := map[string]any{
		"window":            m.Window,
		"source":            m.Source,
		"total":             m.Total,
		"rpm":               m.RPM,
		"error_rate":        m.ErrorRate,
		"p50_ms":            m.P50Ms,
		"p95_ms":            m.P95Ms,
		"cache_hit_rate":    m.CacheHitRate,
		"waf_blocks":        m.WAFBlocks,
		"status_counts":     m.StatusCounts,
		"timeline":          m.Timeline,
		"slowest":           m.Slowest,
		"error_samples":     m.ErrorSamples,
		"latency_histogram": m.LatencyHistogram,
		"top_hosts":         m.TopHosts,
		"top_paths":         m.TopPaths,
		"delta":             a.Delta,
		"upstream":          a.Upstream,
		"compare":           a.Compare,
	}
	if m.RangeFrom != "" {
		out["range_from"] = m.RangeFrom
	}
	if m.RangeTo != "" {
		out["range_to"] = m.RangeTo
	}
	if len(a.PathBreakdown) > 0 {
		out["path_breakdown"] = a.PathBreakdown
	}
	if len(a.ScopeHosts) > 0 {
		out["scope_hosts"] = a.ScopeHosts
	}
	if len(a.ScopePaths) > 0 {
		out["scope_paths"] = a.ScopePaths
	}
	if len(a.WAFTopRules) > 0 {
		out["waf_top_rules"] = a.WAFTopRules
	}
	if len(a.HealthHistory) > 0 {
		out["health_history"] = a.HealthHistory
	}
	if len(a.RelatedRoutes) > 0 {
		out["related_routes"] = a.RelatedRoutes
	}
	if a.RouteCache != nil {
		out["route_cache"] = a.RouteCache
	}
	return out
}

func subscriberRouteIndices(sub *Subscriber) (int, int, bool) {
	if sub == nil {
		return 0, 0, false
	}
	riStr := strings.TrimSpace(sub.Param("ri"))
	piStr := strings.TrimSpace(sub.Param("pi"))
	if riStr == "" || piStr == "" {
		return 0, 0, false
	}
	ri, err := strconv.Atoi(riStr)
	if err != nil {
		return 0, 0, false
	}
	pi, err := strconv.Atoi(piStr)
	if err != nil {
		return 0, 0, false
	}
	return ri, pi, true
}

func subscriberRouteScope(sub *Subscriber) (host, path, pathMatch string) {
	if sub == nil {
		return "", "", "prefix"
	}
	host = strings.TrimSpace(sub.Param("host"))
	path = strings.TrimSpace(sub.Param("path"))
	pathMatch = strings.ToLower(strings.TrimSpace(sub.Param("path_match")))
	if pathMatch == "" {
		pathMatch = "prefix"
	}
	return host, path, pathMatch
}

func routeMetricsCacheKey(sub *Subscriber) string {
	if sub == nil {
		return ""
	}
	ri, pi, ok := subscriberRouteIndices(sub)
	if !ok {
		return ""
	}
	host, path, pathMatch := subscriberRouteScope(sub)
	return subscriberRangeCacheKey(sub) + "|ri:" + strconv.Itoa(ri) + "|pi:" + strconv.Itoa(pi) +
		"|h:" + host + "|p:" + path + "|m:" + pathMatch
}
