package service

import (
	"strings"
)

// ServiceMetricsBuilder builds service detail metrics from access logs.
type ServiceMetricsBuilder struct {
	ingress *Ingress
	health  *HealthCheckService
}

// NewServiceMetricsBuilder creates a service metrics builder.
func NewServiceMetricsBuilder(ingress *Ingress, health *HealthCheckService) *ServiceMetricsBuilder {
	return &ServiceMetricsBuilder{ingress: ingress, health: health}
}

// Build returns analytics for one catalog service.
func (b *ServiceMetricsBuilder) Build(
	window string,
	rangeQ MetricsRangeQuery,
	aliases []string,
) ServiceAnalytics {
	window = strings.TrimSpace(window)
	if window == "" {
		window = WindowLabelForDuration(rangeQ.Duration())
	}

	lines := []string(nil)
	if b != nil && b.ingress != nil && b.ingress.Logs() != nil {
		var err error
		lines, err = b.ingress.Logs().TailAccess(TailLinesForWindow(window))
		if err != nil {
			return BuildServiceAnalytics(window, rangeQ, nil, aliases, b.health)
		}
	}

	health := (*HealthCheckService)(nil)
	if b != nil {
		health = b.health
	}
	return BuildServiceAnalytics(window, rangeQ, lines, aliases, health)
}

// ServiceAnalyticsToMap serializes service analytics for REST/SSE payloads.
func ServiceAnalyticsToMap(a ServiceAnalytics) map[string]any {
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
		"delta":             m.Delta,
		"upstream":          a.Upstream,
		"compare": map[string]any{
			"site_rpm":            a.Compare.SiteRPM,
			"site_error_rate":     a.Compare.SiteErrorRate,
			"service_share_pct":   a.Compare.ServiceSharePct,
			"error_rate_vs_site":  a.Compare.ErrorRateVsSite,
		},
		"target_aliases": a.TargetAliases,
		"health_checks":  a.HealthChecks,
		"health_summary": a.HealthSummary,
	}
	if m.RangeFrom != "" {
		out["range_from"] = m.RangeFrom
	}
	if m.RangeTo != "" {
		out["range_to"] = m.RangeTo
	}
	if len(a.TopHosts) > 0 {
		out["top_hosts"] = a.TopHosts
	}
	if len(a.TopPaths) > 0 {
		out["top_paths"] = a.TopPaths
	}
	return out
}

func subscriberServiceName(sub *Subscriber) string {
	if sub == nil {
		return ""
	}
	return strings.TrimSpace(sub.Param("name"))
}

func serviceMetricsCacheKey(sub *Subscriber) string {
	name := subscriberServiceName(sub)
	if name == "" {
		return ""
	}
	return subscriberRangeCacheKey(sub) + "|name:" + name
}
