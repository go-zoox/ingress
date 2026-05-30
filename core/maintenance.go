package core

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/ingress/core/security"
	"github.com/go-zoox/ingress/core/service"
	"github.com/go-zoox/ingress/core/waf"
	"github.com/go-zoox/zoox"
)

type compiledMaintenanceSettings struct {
	RetryAfter int64
	Title      string
	Subtitle   string
	bypass     compiledMaintenanceBypass
}

type compiledMaintenanceWindow struct {
	configured bool
	start      time.Time
	end        time.Time
}

func (w compiledMaintenanceWindow) activeAt(now time.Time) bool {
	if !w.configured {
		return true
	}
	if !w.start.IsZero() && now.Before(w.start) {
		return false
	}
	if !w.end.IsZero() && now.After(w.end) {
		return false
	}
	return true
}

type compiledMaintenanceHostEntry struct {
	host   waf.SingleHostPattern
	window compiledMaintenanceWindow
}

type compiledMaintenanceHostList struct {
	entries []compiledMaintenanceHostEntry
}

func (l compiledMaintenanceHostList) Len() int {
	return len(l.entries)
}

func (l compiledMaintenanceHostList) MatchesActive(hostname string, now time.Time) bool {
	for _, e := range l.entries {
		if e.host.Matches(hostname) && e.window.activeAt(now) {
			return true
		}
	}
	return false
}

type compiledRuleMaintenance struct {
	enabled  bool
	scopeAll bool
	hosts    compiledMaintenanceHostList
	settings compiledMaintenanceSettings
}

type compiledGlobalMaintenance struct {
	hosts    compiledMaintenanceHostList
	settings compiledMaintenanceSettings
}

type compiledMaintenanceBypass struct {
	allowNets []*net.IPNet
	paths     []maintenanceBypassPath
	headers   []maintenanceBypassHeaderPair
}

type maintenanceBypassHeaderPair struct {
	name  string
	value string
}

type maintenanceBypassPath struct {
	exact  string
	prefix string
}

func compileGlobalMaintenance(cfg MaintenanceConfig) (compiledGlobalMaintenance, error) {
	out := compiledGlobalMaintenance{}
	settings, err := compileMaintenanceSettings(cfg.RetryAfter, cfg.Title, cfg.Subtitle, cfg.Bypass, "maintenance")
	if err != nil {
		return out, err
	}
	out.settings = settings
	if len(cfg.Hosts) > 0 {
		out.hosts, err = compileMaintenanceHostList(cfg.Hosts, "maintenance.hosts")
		if err != nil {
			return out, err
		}
	}
	return out, nil
}

func validateGlobalMaintenance(cfg *Config) error {
	if cfg == nil || !cfg.Maintenance.Configured() {
		return nil
	}
	if _, err := compileGlobalMaintenance(cfg.Maintenance); err != nil {
		return err
	}
	return nil
}

func compileMaintenanceByRule(cfg *Config) ([]compiledRuleMaintenance, error) {
	if cfg == nil {
		return nil, nil
	}
	out := make([]compiledRuleMaintenance, len(cfg.Rules))
	for i := range cfg.Rules {
		r := &cfg.Rules[i]
		m, err := compileServiceMaintenance(r.Backend.Service.Maintenance, ruleBackendLoc(i, r.Host, "/")+".service")
		if err != nil {
			return nil, err
		}
		out[i] = m
	}
	return out, nil
}

func compileServiceMaintenance(m service.Maintenance, loc string) (compiledRuleMaintenance, error) {
	out := compiledRuleMaintenance{enabled: m.Enabled}
	if !m.Configured() {
		return out, nil
	}

	scope := m.EffectiveScope()
	switch scope {
	case service.MaintenanceScopeAll, service.MaintenanceScopeListed:
	default:
		return out, fmt.Errorf("%s.maintenance.scope %q is invalid (all or listed)", loc, m.Scope)
	}
	if m.Enabled && scope == service.MaintenanceScopeListed {
		if len(m.Hosts) == 0 {
			return out, fmt.Errorf("%s.maintenance.hosts is required when scope is listed", loc)
		}
		hosts, err := compileMaintenanceHostList(m.Hosts, loc+".maintenance.hosts")
		if err != nil {
			return out, err
		}
		out.hosts = hosts
	}
	if m.Enabled && scope == service.MaintenanceScopeAll && len(m.Hosts) > 0 {
		return out, fmt.Errorf("%s.maintenance.hosts must not be set when scope is all", loc)
	}
	out.scopeAll = scope == service.MaintenanceScopeAll

	settings, err := compileMaintenanceSettings(m.RetryAfter, m.Title, m.Subtitle, m.Bypass, loc+".maintenance")
	if err != nil {
		return out, err
	}
	out.settings = settings
	return out, nil
}

func compileMaintenanceHostList(entries service.MaintenanceHostList, loc string) (compiledMaintenanceHostList, error) {
	out := compiledMaintenanceHostList{}
	for i, entry := range entries {
		pattern := entry.Pattern()
		if pattern == "" {
			return out, fmt.Errorf("%s[%d].host is required", loc, i)
		}
		host, err := waf.CompileSingleHostPattern(pattern)
		if err != nil {
			return out, fmt.Errorf("%s[%d].host: %w", loc, i, err)
		}
		win, err := compileMaintenanceWindow(entry.Window, fmt.Sprintf("%s[%d].window", loc, i))
		if err != nil {
			return out, err
		}
		out.entries = append(out.entries, compiledMaintenanceHostEntry{host: host, window: win})
	}
	return out, nil
}

func compileMaintenanceWindow(w service.MaintenanceWindow, loc string) (compiledMaintenanceWindow, error) {
	out := compiledMaintenanceWindow{}
	if !w.Configured() {
		return out, nil
	}
	start, err := parseMaintenanceTime(w.Start, loc+".start")
	if err != nil {
		return out, err
	}
	end, err := parseMaintenanceTime(w.End, loc+".end")
	if err != nil {
		return out, err
	}
	if start.IsZero() && end.IsZero() {
		return out, fmt.Errorf("%s requires start and/or end", loc)
	}
	if !start.IsZero() && !end.IsZero() && end.Before(start) {
		return out, fmt.Errorf("%s.end must not be before %s.start", loc, loc)
	}
	out.configured = true
	out.start = start
	out.end = end
	return out, nil
}

func parseMaintenanceTime(raw, loc string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, nil
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("%s %q is invalid (use RFC3339, e.g. 2026-05-30T02:00:00+08:00)", loc, raw)
}

func compileMaintenanceSettings(retryAfter int64, title, subtitle string, bypass service.MaintenanceBypass, loc string) (compiledMaintenanceSettings, error) {
	out := compiledMaintenanceSettings{
		RetryAfter: retryAfter,
		Title:      strings.TrimSpace(title),
		Subtitle:   strings.TrimSpace(subtitle),
	}
	if retryAfter < 0 {
		return out, fmt.Errorf("%s.retry_after must be >= 0", loc)
	}
	b, err := compileMaintenanceBypass(bypass, loc+".bypass")
	if err != nil {
		return out, err
	}
	out.bypass = b
	return out, nil
}

func compileMaintenanceBypass(bypass service.MaintenanceBypass, loc string) (compiledMaintenanceBypass, error) {
	out := compiledMaintenanceBypass{}
	for _, raw := range bypass.AllowIPs {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		_, n, err := parseMaintenanceIPCIDR(raw)
		if err != nil {
			return out, fmt.Errorf("%s.allow_ips: %w", loc, err)
		}
		out.allowNets = append(out.allowNets, n)
	}
	for _, raw := range bypass.Paths {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if strings.Contains(raw, "*") && !strings.HasSuffix(raw, "*") {
			return out, fmt.Errorf("%s.paths: wildcard %q is only supported as a trailing suffix", loc, raw)
		}
		if strings.HasSuffix(raw, "*") {
			out.paths = append(out.paths, maintenanceBypassPath{prefix: strings.TrimSuffix(raw, "*")})
			continue
		}
		out.paths = append(out.paths, maintenanceBypassPath{exact: raw})
	}
	if name := http.CanonicalHeaderKey(strings.TrimSpace(bypass.Header.Name)); name != "" {
		if bypass.Header.Value == "" {
			return out, fmt.Errorf("%s.header.value is required when name is set", loc)
		}
		out.headers = append(out.headers, maintenanceBypassHeaderPair{name: name, value: bypass.Header.Value})
	}
	return out, nil
}

func validateServiceMaintenance(m service.Maintenance, backendType, loc string, hostLevel bool) error {
	if !m.Configured() {
		return nil
	}
	if !hostLevel {
		return fmt.Errorf("%s.maintenance is only supported on rules[].backend.service (host level)", loc)
	}
	if backendType != backendTypeService {
		return fmt.Errorf("%s.maintenance requires backend.type service", loc)
	}
	_, err := compileServiceMaintenance(m, loc)
	return err
}

func (c *core) maintenanceDecision(ruleIdx int, hostname, path string, req *http.Request) (block bool, settings compiledMaintenanceSettings) {
	now := time.Now()
	globalHit := c.globalMaintenance.hosts.MatchesActive(hostname, now)

	ruleHit := false
	var ruleMaint compiledRuleMaintenance
	if ruleIdx >= 0 && ruleIdx < len(c.maintenanceByRule) {
		ruleMaint = c.maintenanceByRule[ruleIdx]
		if ruleMaint.enabled {
			if ruleMaint.scopeAll {
				ruleHit = true
			} else if ruleMaint.hosts.MatchesActive(hostname, now) {
				ruleHit = true
			}
		}
	}

	if !globalHit && !ruleHit {
		return false, compiledMaintenanceSettings{}
	}

	bypass := mergeMaintenanceBypass(c.globalMaintenance.settings.bypass, ruleMaint.settings.bypass)
	if maintenanceBypassAllows(bypass, req, path) {
		return false, compiledMaintenanceSettings{}
	}

	settings = mergeMaintenanceSettings(c.globalMaintenance.settings, ruleMaint.settings, ruleHit)
	return true, settings
}

func mergeMaintenanceBypass(a, b compiledMaintenanceBypass) compiledMaintenanceBypass {
	return compiledMaintenanceBypass{
		allowNets: append(append([]*net.IPNet{}, a.allowNets...), b.allowNets...),
		paths:     append(append([]maintenanceBypassPath{}, a.paths...), b.paths...),
		headers:   append(append([]maintenanceBypassHeaderPair{}, a.headers...), b.headers...),
	}
}

func maintenanceBypassAllows(b compiledMaintenanceBypass, r *http.Request, path string) bool {
	if r == nil {
		return false
	}
	for _, h := range b.headers {
		if h.name != "" && r.Header.Get(h.name) == h.value {
			return true
		}
	}
	if maintenanceBypassPathMatches(b.paths, path) {
		return true
	}
	if len(b.allowNets) > 0 {
		ip := maintenanceClientIP(r)
		if ip != nil && maintenanceIPMatchesNets(ip, b.allowNets) {
			return true
		}
	}
	return false
}

func mergeMaintenanceSettings(global, rule compiledMaintenanceSettings, ruleTriggered bool) compiledMaintenanceSettings {
	out := global
	if !ruleTriggered {
		return out
	}
	if rule.RetryAfter > 0 {
		out.RetryAfter = rule.RetryAfter
	}
	if rule.Title != "" {
		out.Title = rule.Title
	}
	if rule.Subtitle != "" {
		out.Subtitle = rule.Subtitle
	}
	out.bypass = mergeMaintenanceBypass(global.bypass, rule.bypass)
	return out
}

func maintenanceBypassPathMatches(patterns []maintenanceBypassPath, path string) bool {
	for _, p := range patterns {
		if p.exact != "" && path == p.exact {
			return true
		}
		if p.prefix != "" && strings.HasPrefix(path, p.prefix) {
			return true
		}
	}
	return false
}

func (c *core) writeMaintenanceResponse(ctx *zoox.Context, secProf *security.Profile, settings compiledMaintenanceSettings, detail ErrorPageDetail) {
	applySecurityHeaders(ctx, secProf)
	if settings.RetryAfter > 0 {
		ctx.SetHeader("Retry-After", strconv.FormatInt(settings.RetryAfter, 10))
	}
	status := http.StatusServiceUnavailable
	title, subtitle := settings.Title, settings.Subtitle
	if title == "" {
		title, subtitle = builtinErrorPageCopy(status)
	}
	if subtitle == "" && settings.Title != "" {
		_, subtitle = builtinErrorPageCopy(status)
	}

	asJSON := requestPrefersJSON(ctx.Request)
	var body, contentType string
	if asJSON {
		body = ingressErrorPageJSON(status, title, subtitle, c.cfg.ErrorPageExposeDetails, detail.Hostname, detail.Path, detail.Method, detail.Reason)
		contentType = errorPageContentTypeJSON
	} else {
		body = ingressErrorPageHTML(status, title, subtitle, c.cfg.ErrorPageExposeDetails, detail.Hostname, detail.Path, detail.Method, detail.Reason, "maintenance")
		contentType = errorPageContentTypeHTML
	}
	ctx.SetHeader("Content-Type", contentType)
	if asJSON {
		ctx.String(status, body)
		return
	}
	ctx.HTML(status, body)
}

func maintenanceClientIP(r *http.Request) net.IP {
	if r == nil {
		return nil
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		host = strings.TrimSpace(r.RemoteAddr)
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip
	}
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		for _, part := range strings.Split(xff, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if ip := net.ParseIP(part); ip != nil {
				return ip
			}
		}
	}
	return nil
}

func maintenanceIPMatchesNets(ip net.IP, nets []*net.IPNet) bool {
	if ip == nil {
		return false
	}
	for _, n := range nets {
		if n != nil && n.Contains(ip) {
			return true
		}
	}
	return false
}

func parseMaintenanceIPCIDR(s string) (net.IP, *net.IPNet, error) {
	if strings.Contains(s, "/") {
		ip, n, err := net.ParseCIDR(s)
		if err != nil {
			return nil, nil, err
		}
		return ip, n, nil
	}
	ip := net.ParseIP(s)
	if ip == nil {
		return nil, nil, fmt.Errorf("invalid IP or CIDR %q", s)
	}
	if ip4 := ip.To4(); ip4 != nil {
		return ip, &net.IPNet{IP: ip4, Mask: net.CIDRMask(32, 32)}, nil
	}
	return ip, &net.IPNet{IP: ip, Mask: net.CIDRMask(128, 128)}, nil
}

func maintenanceLabelFromRule(r *rule.Rule) string {
	if r == nil || !r.Backend.Service.Maintenance.Enabled {
		return ""
	}
	if r.Backend.Service.Maintenance.EffectiveScope() == service.MaintenanceScopeListed {
		return "partial"
	}
	return "on"
}
