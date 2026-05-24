package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-zoox/ingress/core/admin/service"
	coresvc "github.com/go-zoox/ingress/core/service"
	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/zoox"
)

// RouteDetailHandler serves route detail and per-route metrics.
type RouteDetailHandler struct {
	ingress *service.Ingress
	metrics *service.Metrics
	health  *service.HealthCheckService
}

// NewRouteDetailHandler creates a new route detail handler.
func NewRouteDetailHandler(ingress *service.Ingress, metrics *service.Metrics, health *service.HealthCheckService) *RouteDetailHandler {
	return &RouteDetailHandler{
		ingress: ingress,
		metrics: metrics,
		health:  health,
	}
}

// GetDetail handles GET /api/v1/routes/:ri/:pi
func (h *RouteDetailHandler) GetDetail(ctx *zoox.Context) {
	ri, pi, err := parseRouteIndices(ctx)
	if err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}

	cfg, err := h.ingress.LoadConfig()
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	if ri < 0 || ri >= len(cfg.Rules) {
		fail(ctx, http.StatusNotFound, "rule index out of range")
		return
	}

	r := &cfg.Rules[ri]

	// Determine which backend to use: rule-level (pi=-1) or path-level
	var b rule.Backend
	var pathStr string
	if pi < 0 || pi >= len(r.Paths) {
		// Rule-level backend
		b = r.Backend
		pathStr = "/"
	} else {
		p := &r.Paths[pi]
		b = p.Backend
		pathStr = p.Path
	}

	detail := buildRouteDetail(ri, pi, r, pathStr, b, h.health)
	ok(ctx, detail)
}

// GetMetrics handles GET /api/v1/routes/:ri/:pi/metrics
func (h *RouteDetailHandler) GetMetrics(ctx *zoox.Context) {
	ri, pi, err := parseRouteIndices(ctx)
	if err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}

	cfg, err := h.ingress.LoadConfig()
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	if ri < 0 || ri >= len(cfg.Rules) {
		fail(ctx, http.StatusNotFound, "rule index out of range")
		return
	}

	r := &cfg.Rules[ri]

	// Determine host and path filter
	host := r.Host
	path := "/"
	if pi >= 0 && pi < len(r.Paths) {
		path = r.Paths[pi].Path
	}

	metrics := h.aggregateRouteMetrics(host, path)
	ok(ctx, metrics)
}

// parseRouteIndices extracts rule_index and path_index from URL params.
func parseRouteIndices(ctx *zoox.Context) (int, int, error) {
	riStr := strings.TrimSpace(ctx.Param().Get("ri").String())
	piStr := strings.TrimSpace(ctx.Param().Get("pi").String())

	ri, err := strconv.Atoi(riStr)
	if err != nil {
		return 0, 0, fmtInvalidIndex("rule")
	}
	pi, err := strconv.Atoi(piStr)
	if err != nil {
		return 0, 0, fmtInvalidIndex("path")
	}
	return ri, pi, nil
}

func fmtInvalidIndex(name string) error {
	return &indexError{name: name}
}

type indexError struct {
	name string
}

func (e *indexError) Error() string {
	return "invalid " + e.name + " index"
}

// buildRouteDetail constructs a RouteDetail response from config data.
func buildRouteDetail(ri, pi int, r *rule.Rule, path string, b rule.Backend, healthSvc *service.HealthCheckService) zoox.H {
	host := r.Host
	bt := getBackendTypeLabel(b)
	target := getBackendTarget(b)

	detail := zoox.H{
		"rule_index":  ri,
		"path_index":  pi,
		"host":        host,
		"path":        path,
		"backend": zoox.H{
			"type":            bt,
			"target":         target,
			"service_name":   b.Service.Name,
			"service_port":   b.Service.Port,
			"service_protocol": b.Service.Protocol,
		},
	}

	// Auth info
	if b.Service.Auth.Type != "" {
		enabled := b.Service.Auth.IsEnabled()
		authDetail := zoox.H{
			"type":    b.Service.Auth.Type,
			"enabled": enabled,
			"summary": authSummaryLabel(b.Service.Auth),
		}
		detail["auth"] = authDetail
	} else {
		detail["auth"] = nil
	}

	// Cache info
	if b.Cache.Enabled {
		detail["cache"] = zoox.H{
			"enabled":     true,
			"ttl":         b.Cache.TTL,
			"max_body_kb": b.Cache.MaxBodyBytes / 1024,
			"key_hash":    b.Cache.KeyHash,
		}
	} else {
		detail["cache"] = nil
	}

	// Health check info
	if b.Service.HealthCheck.Enable {
		detail["health_check"] = zoox.H{
			"enabled": true,
			"method":  b.Service.HealthCheck.Method,
			"path":    b.Service.HealthCheck.Path,
			"status":  b.Service.HealthCheck.Status,
			"ok":      b.Service.HealthCheck.Ok,
		}
	} else {
		detail["health_check"] = nil
	}

	// WAF info
	wafDetail := zoox.H{
		"enabled":  false,
		"log_only": false,
		"patched":  len(r.WAFPatch) > 0,
	}
	detail["waf"] = wafDetail

	// If we have health check results, include the status
	if healthSvc != nil && b.Service.HealthCheck.Enable {
		key := host + "|" + path + "|" + target
		if result, ok := healthSvc.GetResult(key); ok {
			detail["health_status"] = result.Status
		}
	}

	return detail
}

// authSummaryLabel generates a short label for auth configuration.
func authSummaryLabel(auth coresvc.Auth) string {
	if auth.Type == "" {
		return ""
	}
	switch auth.Type {
	case "basic":
		return "basic (" + strconv.Itoa(len(auth.Basic.Users)) + " users)"
	case "bearer":
		return "bearer"
	case "oauth2":
		if auth.OAuth2.Provider != "" {
			return "oauth2 (" + auth.OAuth2.Provider + ")"
		}
		return "oauth2"
	default:
		return auth.Type
	}
}

// getBackendTypeLabel determines the backend type string.
func getBackendTypeLabel(b rule.Backend) string {
	if b.Type != "" {
		return b.Type
	}
	if b.Redirect.URL != "" {
		return "redirect"
	}
	if b.Handler.Type != "" {
		return "handler"
	}
	return "service"
}

// getBackendTarget returns the backend target summary string.
func getBackendTarget(b rule.Backend) string {
	switch getBackendTypeLabel(b) {
	case "redirect":
		if b.Redirect.URL != "" {
			return b.Redirect.URL
		}
		return "(redirect)"
	case "handler":
		return b.Handler.Type
	default:
		s := b.Service
		if s.Name == "" {
			return ""
		}
		port := s.Port
		if port == 0 {
			if s.Protocol == "https" {
				port = 443
			} else {
				port = 80
			}
		}
		return s.Name + ":" + strconv.FormatInt(port, 10)
	}
}

// aggregateRouteMetrics computes metrics filtered by host and path.
func (h *RouteDetailHandler) aggregateRouteMetrics(host, path string) zoox.H {
	lines, err := h.ingress.Logs().TailAccess(5000)
	if err != nil || len(lines) == 0 {
		return zoox.H{
			"window":         "15m",
			"rpm":            0,
			"error_rate":     0,
			"p50_ms":         0,
			"p95_ms":         0,
			"cache_hit_rate": 0,
			"total":          0,
			"timeline":       []interface{}{},
		}
	}

	// Parse and filter access log entries by host and path
	var entries []service.AccessEntry
	for _, line := range lines {
		e, ok := service.ParseAccessEntry(line)
		if !ok {
			continue
		}
		// Filter by host (case-insensitive)
		if host != "" && !strings.EqualFold(e.Host, host) {
			continue
		}
		// Filter by path (prefix match)
		if path != "" && path != "/" && !strings.HasPrefix(e.Path, path) {
			continue
		}
		entries = append(entries, e)
	}

	total := len(entries)
	if total == 0 {
		return zoox.H{
			"window":         "15m",
			"rpm":            0,
			"error_rate":     0,
			"p50_ms":         0,
			"p95_ms":         0,
			"cache_hit_rate": 0,
			"total":          0,
			"timeline":       []interface{}{},
		}
	}

	// Aggregate metrics
	errorCount := 0
	cacheHits := 0
	var durations []float64

	for _, e := range entries {
		if e.Status >= 400 {
			errorCount++
		}
		if e.CacheHit {
			cacheHits++
		}
		if e.DurationMs > 0 {
			durations = append(durations, e.DurationMs)
		}
	}

	errorRate := float64(errorCount) / float64(total) * 100
	cacheHitRate := float64(cacheHits) / float64(total) * 100
	rpm := float64(total) / 15.0 // 15m window

	var p50, p95 float64
	if ps := service.ComputePercentiles(durations, 0.5, 0.95); len(ps) >= 2 {
		p50, p95 = ps[0], ps[1]
	}

	return zoox.H{
		"window":         "15m",
		"rpm":            rpm,
		"error_rate":     errorRate,
		"p50_ms":         p50,
		"p95_ms":         p95,
		"cache_hit_rate": cacheHitRate,
		"total":          total,
		"timeline":       []interface{}{},
	}
}
