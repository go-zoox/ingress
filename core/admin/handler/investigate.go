package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-zoox/ingress/core/admin/service"
	ingcore "github.com/go-zoox/ingress/core"
	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/zoox"
)

// Investigate handles GET /api/v1/investigate
func (a *API) Investigate(ctx *zoox.Context) {
	host := strings.TrimSpace(ctx.Query().Get("host").String())
	path := strings.TrimSpace(ctx.Query().Get("path").String())
	if path == "" {
		path = "/"
	}
	if host == "" {
		fail(ctx, http.StatusBadRequest, "host is required")
		return
	}

	method := strings.TrimSpace(ctx.Query().Get("method").String())
	limit := 20
	if v := strings.TrimSpace(ctx.Query().Get("limit").String()); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}

	ri, pi := -1, -1
	if v := strings.TrimSpace(ctx.Query().Get("ri").String()); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			ri = n
		}
	}
	if v := strings.TrimSpace(ctx.Query().Get("pi").String()); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			pi = n
		}
	}

	icfg, err := a.ingress.LoadConfig()
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	preview, useRI, usePI, err := service.MatchForInvestigate(icfg, host, path, ri, pi)
	if err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}

	if preview == nil && useRI >= 0 && usePI >= 0 {
		if useRI < len(icfg.Rules) {
			r := &icfg.Rules[useRI]
			p := path
			if usePI >= 0 && usePI < len(r.Paths) {
				p = r.Paths[usePI].Path
			}
			preview = &ingcore.MatchPreview{
				Matched:     true,
				RuleIndex:   useRI,
				PathIndex:   usePI,
				Host:        r.Host,
				HostType:    r.HostType,
				Path:        p,
				BackendType: "",
			}
		}
	}

	var route zoox.H
	if useRI >= 0 && usePI >= 0 && useRI < len(icfg.Rules) {
		r := &icfg.Rules[useRI]
		var b rule.Backend
		var pathStr string
		if usePI < 0 || usePI >= len(r.Paths) {
			b = r.Backend
			pathStr = "/"
		} else {
			b = r.Paths[usePI].Backend
			pathStr = r.Paths[usePI].Path
		}
		route = buildRouteDetail(useRI, usePI, r, pathStr, b, a.health)
		if preview != nil && preview.Matched {
			preview.BackendType = getBackendTypeLabel(b)
			preview.Target = getBackendTarget(b)
		}
	}

	lines, _ := a.logs.TailAccess(5000)
	entries := service.FilterAccessEntries(lines, host, path, method, limit)
	samples := service.EntriesToSamples(entries)
	stats := service.StatsFromEntries(entries)

	wafRows, _ := a.audit.ListWAFEvents(service.WAFAuditFilter{
		Action: "block",
		Host:   host,
		Path:   path,
		Limit:  8,
	})

	var healthChecks []service.HealthCheckResult
	if a.health != nil {
		checks, _ := a.health.ListResults()
		healthChecks = service.FilterHealthChecks(checks, host, path)
	}

	ok(ctx, zoox.H{
		"query": zoox.H{
			"host":   host,
			"path":   path,
			"method": method,
		},
		"match":       preview,
		"route":       route,
		"samples":     samples,
		"stats":       stats,
		"waf_recent":  wafRows,
		"health_checks": healthChecks,
	})
}
