package service

import (
	"strconv"
	"strings"
	"time"

	"github.com/go-zoox/ingress/core/admin/model"
	ingcore "github.com/go-zoox/ingress/core"
	"github.com/go-zoox/ingress/core/rule"
)

// RouteAnalytics extends overview metrics with route-detail panels (P1–P3).
type RouteAnalytics struct {
	OverviewMetrics
	Delta         OverviewDelta        `json:"delta"`
	Upstream      UpstreamLatencyStats `json:"upstream"`
	PathBreakdown []PathBreakdownRow   `json:"path_breakdown,omitempty"`
	ScopeHosts       []NamedCount       `json:"scope_hosts,omitempty"`
	ScopePaths       []NamedCount       `json:"scope_paths,omitempty"`
	ScopeHostTraffic []HostTrafficStat  `json:"scope_host_traffic,omitempty"`
	WAFTopRules      []NamedCount       `json:"waf_top_rules,omitempty"`
	HealthHistory []HealthProbePoint   `json:"health_history,omitempty"`
	Compare       RouteCompareStats    `json:"compare"`
	RelatedRoutes []RelatedRouteRow    `json:"related_routes,omitempty"`
	RouteCache    *RouteCacheStats     `json:"route_cache,omitempty"`
}

// UpstreamLatencyStats splits total request time into gateway vs upstream portions.
type UpstreamLatencyStats struct {
	Samples          int     `json:"samples"`
	AvgTotalMs       float64 `json:"avg_total_ms"`
	AvgUpstreamMs    float64 `json:"avg_upstream_ms"`
	AvgGatewayMs     float64 `json:"avg_gateway_ms"`
	UpstreamErrorPct float64 `json:"upstream_error_pct"`
}

// PathBreakdownRow is per-path traffic within a host rule.
type PathBreakdownRow struct {
	PathIndex int     `json:"path_index"`
	Path      string  `json:"path"`
	Count     int     `json:"count"`
	ErrorRate float64 `json:"error_rate"`
}

// RouteCompareStats compares the route to site-wide traffic in the same window.
type RouteCompareStats struct {
	SiteRPM         float64 `json:"site_rpm"`
	SiteErrorRate   float64 `json:"site_error_rate"`
	RouteSharePct   float64 `json:"route_share_pct"`
	ErrorRateVsSite float64 `json:"error_rate_vs_site"`
}

// RelatedRouteRow links to other routes sharing backend or host suffix.
type RelatedRouteRow struct {
	RuleIndex int    `json:"rule_index"`
	PathIndex int    `json:"path_index"`
	Host      string `json:"host"`
	Path      string `json:"path"`
	Target    string `json:"target"`
	Relation  string `json:"relation"`
}

// RouteCacheStats is HTTP response cache stats for this route.
type RouteCacheStats struct {
	Enabled   bool    `json:"enabled"`
	TTL       int64   `json:"ttl"`
	MaxBodyKB int64   `json:"max_body_kb"`
	Hits      int     `json:"hits"`
	Total     int     `json:"total"`
	HitRate   float64 `json:"hit_rate"`
}

// HealthProbePoint is one health-check sample for timeline charts.
type HealthProbePoint struct {
	At         string  `json:"at"`
	Status     string  `json:"status"`
	ResponseMs float64 `json:"response_ms"`
}

// applyAccessEntryRange filters parsed entries to the metrics interval.
func applyAccessEntryRange(entries []AccessEntry, window string, rangeQ MetricsRangeQuery) []AccessEntry {
	if !rangeQ.From.IsZero() && !rangeQ.To.IsZero() {
		return filterEntriesSince(entries, rangeQ.From, rangeQ.To)
	}
	return entriesInMetricsWindow(entries, window)
}

// BuildRouteAnalytics aggregates access logs and auxiliary data for one route.
func BuildRouteAnalytics(
	cfg *ingcore.Config,
	ruleIndex, pathIndex int,
	window string,
	rangeQ MetricsRangeQuery,
	lines []string,
	health *HealthCheckService,
	wafEvents []model.WAFEvent,
	cacheEnabled bool,
	cacheTTL, cacheMaxBodyKB int64,
	scopeHost, scopePath string,
	pathMatch string,
) RouteAnalytics {
	window = strings.TrimSpace(window)
	if window == "" {
		window = "15m"
	}
	if !rangeQ.From.IsZero() && !rangeQ.To.IsZero() {
		window = WindowLabelForDuration(rangeQ.Duration())
	}
	source := "access_log"

	all := make([]AccessEntry, 0, len(lines))
	parseResult := ParseAccessLogLines(lines)
	all = parseResult.Entries
	if len(lines) == 0 {
		source = "access_log_empty"
	} else if len(all) == 0 && parseResult.IssueSkipped == 0 {
		source = "access_log_empty"
	}

	routeEntries := FilterAccessEntriesForRoute(cfg, ruleIndex, pathIndex, lines)
	all = applyAccessEntryRange(all, window, rangeQ)
	routeEntries = applyAccessEntryRange(routeEntries, window, rangeQ)
	scopeHosts, scopePaths := ScopeHostPathCounts(routeEntries, window)
	scopeHostTraffic := hostTrafficStats(routeEntries, -1)
	filtered := filterAccessEntriesByScope(routeEntries, scopeHost, scopePath, pathMatch)
	overview := AggregateAccessEntries(filtered, window, source)
	if !rangeQ.From.IsZero() && !rangeQ.To.IsZero() {
		overview.Window = "range"
		overview.RangeFrom = rangeQ.From.Format(time.RFC3339)
		overview.RangeTo = rangeQ.To.Format(time.RFC3339)
	}
	site := AggregateAccessEntries(all, window, source)

	windowDur := parseWindowDuration(window)
	hasTime := entriesHaveTimestamps(all)
	anchor := latestEntryTime(all)
	if anchor.IsZero() {
		anchor = latestEntryTime(filtered)
	}
	prevFiltered := filterEntriesInPreviousWindow(filtered, anchor, windowDur, hasTime)
	delta := computeOverviewDelta(filtered, prevFiltered, windowDur)
	overview.Delta = delta

	routeShare := 0.0
	if site.Total > 0 {
		routeShare = float64(overview.Total) / float64(site.Total) * 100
	}

	// WAF Top Rules should also respect the same scope filters.
	wafEvents = filterWAFEventsByScope(wafEvents, scopeHost, scopePath, pathMatch)
	out := RouteAnalytics{
		OverviewMetrics: overview,
		Delta:           delta,
		Upstream:        buildUpstreamStats(filtered),
		PathBreakdown:   buildPathBreakdown(cfg, ruleIndex, filtered),
		ScopeHosts:       scopeHosts,
		ScopePaths:       scopePaths,
		ScopeHostTraffic: scopeHostTraffic,
		WAFTopRules:     topWAFRulesFromEvents(wafEvents, 8),
		Compare: RouteCompareStats{
			SiteRPM:         site.RPM,
			SiteErrorRate:   site.ErrorRate,
			RouteSharePct:   routeShare,
			ErrorRateVsSite: overview.ErrorRate - site.ErrorRate,
		},
		RelatedRoutes: listRelatedRoutes(cfg, ruleIndex, pathIndex),
	}
	if cacheEnabled {
		out.RouteCache = buildRouteCacheStats(filtered, cacheTTL, cacheMaxBodyKB)
	}
	if health != nil {
		key := healthKeyForRoute(cfg, ruleIndex, pathIndex)
		if key != "" {
			out.HealthHistory = health.GetHistory(key)
		}
	}
	return out
}

func topWAFRulesFromEvents(events []model.WAFEvent, n int) []NamedCount {
	counts := map[string]int{}
	for _, e := range events {
		rule := strings.TrimSpace(e.Rule)
		if rule == "" {
			continue
		}
		counts[rule]++
	}
	return topN(counts, n)
}

func filterAccessEntriesByScope(entries []AccessEntry, scopeHost, scopePath, pathMatch string) []AccessEntry {
	if len(entries) == 0 {
		return entries
	}
	if strings.TrimSpace(scopeHost) == "" && strings.TrimSpace(scopePath) == "" {
		return entries
	}

	out := make([]AccessEntry, 0, len(entries))
	for _, e := range entries {
		if scopeHost != "" && !strings.EqualFold(strings.TrimSpace(e.Host), strings.TrimSpace(scopeHost)) {
			continue
		}
		if scopePath != "" && !MatchPathForScope(e.Path, scopePath, pathMatch) {
			continue
		}
		out = append(out, e)
	}
	return out
}

func filterWAFEventsByScope(events []model.WAFEvent, scopeHost, scopePath, pathMatch string) []model.WAFEvent {
	if len(events) == 0 {
		return events
	}
	if strings.TrimSpace(scopeHost) == "" && strings.TrimSpace(scopePath) == "" {
		return events
	}

	out := make([]model.WAFEvent, 0, len(events))
	for _, e := range events {
		if scopeHost != "" && !strings.EqualFold(strings.TrimSpace(e.Host), strings.TrimSpace(scopeHost)) {
			continue
		}
		if scopePath != "" && !MatchPathForScope(e.Path, scopePath, pathMatch) {
			continue
		}
		out = append(out, e)
	}
	return out
}

func buildUpstreamStats(entries []AccessEntry) UpstreamLatencyStats {
	var st UpstreamLatencyStats
	var totalSum, upSum, gwSum float64
	upErrors := 0
	for _, e := range entries {
		if e.DurationMs <= 0 {
			continue
		}
		st.Samples++
		totalSum += e.DurationMs
		up := e.UpstreamDurationMs
		if up <= 0 {
			up = e.DurationMs
		}
		upSum += up
		gw := e.DurationMs - up
		if gw < 0 {
			gw = 0
		}
		gwSum += gw
		us := e.UpstreamStatus
		if us == 0 {
			us = e.Status
		}
		if us >= 500 {
			upErrors++
		}
	}
	if st.Samples == 0 {
		return st
	}
	n := float64(st.Samples)
	st.AvgTotalMs = totalSum / n
	st.AvgUpstreamMs = upSum / n
	st.AvgGatewayMs = gwSum / n
	st.UpstreamErrorPct = float64(upErrors) / n * 100
	return st
}

func buildPathBreakdown(cfg *ingcore.Config, ruleIndex int, entries []AccessEntry) []PathBreakdownRow {
	if cfg == nil || ruleIndex < 0 || ruleIndex >= len(cfg.Rules) {
		return nil
	}
	r := &cfg.Rules[ruleIndex]
	if len(r.Paths) == 0 {
		return nil
	}
	type scratch struct {
		count  int
		errors int
	}
	scratches := make([]scratch, len(r.Paths))
	var defaultScr scratch
	for _, e := range entries {
		pi, err := ingcore.MatchPathIndexForRule(cfg, ruleIndex, e.Host, e.Path)
		if err != nil || pi < 0 {
			defaultScr.count++
			if e.Status >= 400 {
				defaultScr.errors++
			}
			continue
		}
		scratches[pi].count++
		if e.Status >= 400 {
			scratches[pi].errors++
		}
	}
	var out []PathBreakdownRow
	if defaultScr.count > 0 {
		out = append(out, pathBreakdownRow(-1, "/", defaultScr.count, defaultScr.errors))
	}
	for i, p := range r.Paths {
		if scratches[i].count == 0 {
			continue
		}
		out = append(out, pathBreakdownRow(i, p.Path, scratches[i].count, scratches[i].errors))
	}
	return out
}

func pathBreakdownRow(pi int, path string, count, errors int) PathBreakdownRow {
	rate := 0.0
	if count > 0 {
		rate = float64(errors) / float64(count) * 100
	}
	return PathBreakdownRow{PathIndex: pi, Path: path, Count: count, ErrorRate: rate}
}

func buildRouteCacheStats(entries []AccessEntry, ttl, maxBodyKB int64) *RouteCacheStats {
	hits := 0
	for _, e := range entries {
		if e.CacheHit {
			hits++
		}
	}
	total := len(entries)
	rate := 0.0
	if total > 0 {
		rate = float64(hits) / float64(total) * 100
	}
	return &RouteCacheStats{
		Enabled:   true,
		TTL:       ttl,
		MaxBodyKB: maxBodyKB,
		Hits:      hits,
		Total:     total,
		HitRate:   rate,
	}
}

func listRelatedRoutes(cfg *ingcore.Config, ruleIndex, pathIndex int) []RelatedRouteRow {
	if cfg == nil || ruleIndex < 0 || ruleIndex >= len(cfg.Rules) {
		return nil
	}
	selfHost := cfg.Rules[ruleIndex].Host
	selfTarget := routeTargetAt(cfg, ruleIndex, pathIndex)
	seen := map[string]struct{}{}
	var out []RelatedRouteRow
	add := func(ri, pi int, host, path, target, rel string) {
		if ri == ruleIndex && pi == pathIndex {
			return
		}
		key := relatedRouteKey(ri, pi)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, RelatedRouteRow{
			RuleIndex: ri, PathIndex: pi, Host: host, Path: path, Target: target, Relation: rel,
		})
	}
	rows, err := ingcore.ListRouteRows(cfg)
	if err != nil {
		return nil
	}
	for _, row := range rows {
		if selfTarget != "" && row.Target == selfTarget {
			add(row.RuleIndex, row.PathIndex, row.Host, row.Path, row.Target, "same_backend")
		}
		if pathIndex < 0 && row.RuleIndex != ruleIndex && hostSharesSuffix(selfHost, row.Host) {
			add(row.RuleIndex, row.PathIndex, row.Host, row.Path, row.Target, "same_host_suffix")
		}
	}
	if len(out) > 6 {
		out = out[:6]
	}
	return out
}

func relatedRouteKey(ri, pi int) string {
	return strconv.Itoa(ri) + ":" + strconv.Itoa(pi)
}

func hostSharesSuffix(a, b string) bool {
	a = strings.ToLower(strings.TrimSpace(a))
	b = strings.ToLower(strings.TrimSpace(b))
	if a == "" || b == "" || a == b {
		return false
	}
	parts := strings.SplitN(a, ".", 2)
	if len(parts) < 2 {
		return false
	}
	suffix := "." + parts[1]
	return strings.HasSuffix(b, suffix)
}

func routeTargetAt(cfg *ingcore.Config, ruleIndex, pathIndex int) string {
	if ruleIndex < 0 || ruleIndex >= len(cfg.Rules) {
		return ""
	}
	r := &cfg.Rules[ruleIndex]
	if pathIndex >= 0 && pathIndex < len(r.Paths) {
		return backendTargetLabel(r.Paths[pathIndex].Backend)
	}
	return backendTargetLabel(r.Backend)
}

func healthKeyForRoute(cfg *ingcore.Config, ruleIndex, pathIndex int) string {
	if cfg == nil || ruleIndex < 0 || ruleIndex >= len(cfg.Rules) {
		return ""
	}
	r := &cfg.Rules[ruleIndex]
	path := "/"
	var b rule.Backend
	if pathIndex >= 0 && pathIndex < len(r.Paths) {
		path = r.Paths[pathIndex].Path
		b = r.Paths[pathIndex].Backend
	} else {
		b = r.Backend
	}
	target := backendTargetLabel(b)
	if target == "" {
		return ""
	}
	return r.Host + "|" + path + "|" + target
}
