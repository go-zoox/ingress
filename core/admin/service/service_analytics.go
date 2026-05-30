package service

import (
	"strings"
)

// ServiceCompareStats compares service-scoped traffic to the whole site.
type ServiceCompareStats struct {
	SiteRPM           float64 `json:"site_rpm"`
	SiteErrorRate     float64 `json:"site_error_rate"`
	ServiceSharePct   float64 `json:"service_share_pct"`
	ErrorRateVsSite   float64 `json:"error_rate_vs_site"`
}

// ServiceAnalytics is metrics for one upstream service (by access-log target).
type ServiceAnalytics struct {
	OverviewMetrics
	Upstream       UpstreamLatencyStats  `json:"upstream"`
	Compare        ServiceCompareStats   `json:"compare"`
	TopHosts       []NamedCount          `json:"top_hosts,omitempty"`
	TopPaths       []NamedCount          `json:"top_paths,omitempty"`
	HealthChecks   []HealthCheckResult   `json:"health_checks,omitempty"`
	HealthSummary  HealthSummary         `json:"health_summary"`
	TargetAliases  []string              `json:"target_aliases,omitempty"`
}

// FilterAccessEntriesForService keeps lines whose upstream target matches any alias.
func FilterAccessEntriesForService(entries []AccessEntry, targets []string) []AccessEntry {
	set := targetSet(targets)
	if len(set) == 0 {
		return nil
	}
	out := make([]AccessEntry, 0, len(entries))
	for _, e := range entries {
		if _, ok := set[strings.TrimSpace(e.Target)]; ok {
			out = append(out, e)
		}
	}
	return out
}

// BuildServiceAnalytics aggregates access-log metrics for upstream target aliases.
func BuildServiceAnalytics(
	window string,
	lines []string,
	targets []string,
	health *HealthCheckService,
) ServiceAnalytics {
	window = strings.TrimSpace(window)
	if window == "" {
		window = "15m"
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

	filtered := FilterAccessEntriesForService(all, targets)
	overview := AggregateAccessEntries(filtered, window, source)
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

	share := 0.0
	if site.Total > 0 {
		share = float64(overview.Total) / float64(site.Total) * 100
	}

	hostCounts := map[string]int{}
	pathCounts := map[string]int{}
	for _, e := range filtered {
		if h := strings.TrimSpace(e.Host); h != "" {
			hostCounts[h]++
		}
		if p := strings.TrimSpace(e.Path); p != "" {
			pathCounts[p]++
		}
	}

	out := ServiceAnalytics{
		OverviewMetrics: overview,
		Upstream:        buildUpstreamStats(filtered),
		Compare: ServiceCompareStats{
			SiteRPM:         site.RPM,
			SiteErrorRate:   site.ErrorRate,
			ServiceSharePct: share,
			ErrorRateVsSite: overview.ErrorRate - site.ErrorRate,
		},
		TopHosts:      topN(hostCounts, 10),
		TopPaths:      topN(pathCounts, 10),
		TargetAliases: append([]string(nil), targets...),
	}

	if health != nil {
		checks, _ := health.ListResults()
		set := targetSet(targets)
		matched := make([]HealthCheckResult, 0)
		for _, c := range checks {
			if _, ok := set[strings.TrimSpace(c.Backend)]; ok {
				matched = append(matched, c)
			}
		}
		out.HealthChecks = matched
		if len(matched) > 0 {
			up, down, unknown := 0, 0, 0
			for _, c := range matched {
				switch c.Status {
				case "up":
					up++
				case "down":
					down++
				default:
					unknown++
				}
			}
			out.HealthSummary = HealthSummary{Total: len(matched), Up: up, Down: down, Unknown: unknown}
		}
	}

	return out
}
