package service

import (
	"strings"

	ingcore "github.com/go-zoox/ingress/core"
)

// InvestigateQuery parameters for request investigation.
type InvestigateQuery struct {
	Host   string
	Path   string
	Method string
	Limit  int
	RI     int // rule index; negative = unset
	PI     int // path index; negative = unset
}

// InvestigateSample is one access log row in an investigation response.
type InvestigateSample struct {
	At                   string  `json:"at,omitempty"`
	ClientIP             string  `json:"client_ip,omitempty"`
	Method               string  `json:"method"`
	Path                 string  `json:"path"`
	Status               int     `json:"status"`
	DurationMs           float64 `json:"duration_ms"`
	Target               string  `json:"target,omitempty"`
	UpstreamStatus       int     `json:"upstream_status,omitempty"`
	UpstreamDurationMs   float64 `json:"upstream_duration_ms,omitempty"`
	CacheHit             bool    `json:"cache_hit"`
	WAFBlock             bool    `json:"waf_block"`
}

// InvestigateStats summarizes filtered samples.
type InvestigateStats struct {
	Count        int     `json:"count"`
	ErrorRate    float64 `json:"error_rate"`
	P95Ms        float64 `json:"p95_ms"`
	CacheHitRate float64 `json:"cache_hit_rate"`
}

// FilterAccessEntries returns entries matching host/path with optional method filter.
func FilterAccessEntries(lines []string, host, path, method string, limit int) []AccessEntry {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	hostLower := strings.ToLower(strings.TrimSpace(host))
	path = strings.TrimSpace(path)
	if path == "" {
		path = "/"
	}
	method = strings.ToUpper(strings.TrimSpace(method))

	var out []AccessEntry
	for i := len(lines) - 1; i >= 0 && len(out) < limit*3; i-- {
		e, ok := parseAccessLine(lines[i])
		if !ok {
			continue
		}
		if strings.ToLower(e.Host) != hostLower {
			continue
		}
		if !pathPrefixMatch(e.Path, path) {
			continue
		}
		if method != "" && e.Method != method {
			continue
		}
		out = append(out, e)
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func pathPrefixMatch(samplePath, filterPath string) bool {
	if filterPath == "" || filterPath == "/" {
		return true
	}
	return samplePath == filterPath || strings.HasPrefix(samplePath, filterPath)
}

// StatsFromEntries computes error rate, p95, cache hit rate from entries.
func StatsFromEntries(entries []AccessEntry) InvestigateStats {
	if len(entries) == 0 {
		return InvestigateStats{}
	}
	errors := 0
	cacheHits := 0
	var durations []float64
	for _, e := range entries {
		if e.Status >= 400 {
			errors++
		}
		if e.CacheHit {
			cacheHits++
		}
		if e.DurationMs > 0 {
			durations = append(durations, e.DurationMs)
		}
	}
	n := float64(len(entries))
	p95 := 0.0
	if ps := ComputePercentiles(durations, 0.95); len(ps) > 0 {
		p95 = ps[0]
	}
	return InvestigateStats{
		Count:        len(entries),
		ErrorRate:    float64(errors) / n * 100,
		P95Ms:        p95,
		CacheHitRate: float64(cacheHits) / n * 100,
	}
}

// EntriesToSamples converts AccessEntry slice to API samples (newest first preserved).
func EntriesToSamples(entries []AccessEntry) []InvestigateSample {
	out := make([]InvestigateSample, 0, len(entries))
	for _, e := range entries {
		s := InvestigateSample{
			ClientIP:           e.ClientIP,
			Method:             e.Method,
			Path:               e.Path,
			Status:             e.Status,
			DurationMs:         e.DurationMs,
			Target:             e.Target,
			UpstreamStatus:     e.UpstreamStatus,
			UpstreamDurationMs: e.UpstreamDurationMs,
			CacheHit:           e.CacheHit,
			WAFBlock:           e.WAFBlock,
		}
		if !e.At.IsZero() {
			s.At = e.At.Format("2006-01-02T15:04:05")
		}
		out = append(out, s)
	}
	return out
}

// FilterHealthChecks returns checks for a host/path.
func FilterHealthChecks(checks []HealthCheckResult, host, path string) []HealthCheckResult {
	hostLower := strings.ToLower(strings.TrimSpace(host))
	path = strings.TrimSpace(path)
	var out []HealthCheckResult
	for _, c := range checks {
		if strings.ToLower(c.Host) != hostLower {
			continue
		}
		if path != "" && path != "/" && c.Path != "" {
			if c.Path != path && !strings.HasPrefix(path, c.Path) && !strings.HasPrefix(c.Path, path) {
				continue
			}
		}
		out = append(out, c)
	}
	return out
}

// MatchForInvestigate runs PreviewMatch unless rule index ri is provided (pi may be -1).
func MatchForInvestigate(cfg *ingcore.Config, host, path string, ri, pi int) (*ingcore.MatchPreview, int, int, error) {
	if ri >= 0 {
		return nil, ri, pi, nil
	}
	preview, err := ingcore.PreviewMatch(cfg, host, path)
	if err != nil {
		return nil, -1, -1, err
	}
	if preview != nil && preview.Matched {
		return preview, preview.RuleIndex, preview.PathIndex, nil
	}
	return preview, -1, -1, nil
}
