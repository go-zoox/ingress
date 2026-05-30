package handler

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/go-zoox/ingress/core/admin/service"
	"github.com/go-zoox/zoox"
)

// ServiceDetailHandler serves catalog service detail and metrics.
type ServiceDetailHandler struct {
	ingress *service.Ingress
	health  *service.HealthCheckService
}

// NewServiceDetailHandler creates a service detail handler.
func NewServiceDetailHandler(ingress *service.Ingress, health *service.HealthCheckService) *ServiceDetailHandler {
	return &ServiceDetailHandler{ingress: ingress, health: health}
}

// GetDetail handles GET /api/v1/services/:name
func (h *ServiceDetailHandler) GetDetail(ctx *zoox.Context) {
	name, err := parseServiceNameParam(ctx)
	if err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}

	content, err := h.ingress.ReadYAML()
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	catalog, err := service.ParseServiceCatalog(content)
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	entry, found := service.FindCatalogService(catalog, name)
	if !found {
		fail(ctx, http.StatusNotFound, "service not found in catalog")
		return
	}

	cfg, err := h.ingress.LoadConfig()
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	refs := service.ListServiceRouteRefs(cfg, name)
	aliases := service.ServiceTargetAliases(entry, refs)

	var healthDetail zoox.H
	if entry.HealthCheck.Enable {
		healthDetail = zoox.H{
			"enabled": true,
			"method":  entry.HealthCheck.Method,
			"path":    entry.HealthCheck.Path,
			"status":  entry.HealthCheck.Status,
			"ok":      entry.HealthCheck.Ok,
		}
	}

	ok(ctx, zoox.H{
		"name":             entry.Name,
		"catalog_index":    entry.Index,
		"target":           entry.Target,
		"protocol":         entry.Protocol,
		"port":             entry.Port,
		"mode":             entry.Mode,
		"note":             entry.Note,
		"health_check":     healthDetail,
		"route_refs":       refs,
		"route_ref_count":  len(refs),
		"target_aliases":   aliases,
	})
}

// GetMetrics handles GET /api/v1/services/:name/metrics
func (h *ServiceDetailHandler) GetMetrics(ctx *zoox.Context) {
	name, err := parseServiceNameParam(ctx)
	if err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}

	content, err := h.ingress.ReadYAML()
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	catalog, err := service.ParseServiceCatalog(content)
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	entry, found := service.FindCatalogService(catalog, name)
	if !found {
		fail(ctx, http.StatusNotFound, "service not found in catalog")
		return
	}

	cfg, err := h.ingress.LoadConfig()
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	refs := service.ListServiceRouteRefs(cfg, name)
	aliases := service.ServiceTargetAliases(entry, refs)

	window := strings.TrimSpace(ctx.Query().Get("window").String())
	lines := []string(nil)
	if logs := h.ingress.Logs(); logs != nil {
		lines, err = logs.TailAccess(service.TailLinesForWindow(window))
		if err != nil {
			fail(ctx, http.StatusInternalServerError, err.Error())
			return
		}
	}

	analytics := service.BuildServiceAnalytics(window, lines, aliases, h.health)
	ok(ctx, serviceAnalyticsJSON(analytics))
}

func parseServiceNameParam(ctx *zoox.Context) (string, error) {
	raw := strings.TrimSpace(ctx.Param().Get("name").String())
	if raw == "" {
		return "", &serviceNameError{}
	}
	decoded, err := url.PathUnescape(raw)
	if err != nil {
		return raw, nil
	}
	return strings.TrimSpace(decoded), nil
}

type serviceNameError struct{}

func (e *serviceNameError) Error() string {
	return "service name is required"
}

func serviceAnalyticsJSON(a service.ServiceAnalytics) zoox.H {
	m := a.OverviewMetrics
	out := zoox.H{
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
		"compare": zoox.H{
			"site_rpm":            a.Compare.SiteRPM,
			"site_error_rate":     a.Compare.SiteErrorRate,
			"service_share_pct":   a.Compare.ServiceSharePct,
			"error_rate_vs_site":  a.Compare.ErrorRateVsSite,
		},
		"target_aliases":  a.TargetAliases,
		"health_checks":   a.HealthChecks,
		"health_summary":  a.HealthSummary,
	}
	if len(a.TopHosts) > 0 {
		out["top_hosts"] = a.TopHosts
	}
	if len(a.TopPaths) > 0 {
		out["top_paths"] = a.TopPaths
	}
	return out
}
