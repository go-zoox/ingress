package core

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/ingress/core/service"
)

const (
	testMaintWindowStart = "2000-01-01T00:00:00Z"
	testMaintWindowEnd   = "2099-12-31T23:59:59Z"
)

func hosts(entries ...service.MaintenanceHostEntry) service.MaintenanceHostList {
	return service.MaintenanceHostList(entries)
}

func maintHost(host string) service.MaintenanceHostEntry {
	return service.MaintenanceHostEntry{
		Host: host,
		Window: service.MaintenanceWindow{
			Start: testMaintWindowStart,
			End:   testMaintWindowEnd,
		},
	}
}

func maintRuleWindow() service.MaintenanceWindow {
	return service.MaintenanceWindow{Start: testMaintWindowStart, End: testMaintWindowEnd}
}

func TestValidateServiceMaintenance_RejectsPathLevel(t *testing.T) {
	err := validateServiceMaintenance(
		service.Maintenance{Enabled: true, Scope: service.MaintenanceScopeAll, Window: maintRuleWindow()},
		backendTypeService,
		"rules[0].paths[0].backend.service",
		false,
	)
	if err == nil {
		t.Fatal("expected path-level maintenance to be rejected")
	}
}

func TestValidateServiceMaintenance_ScopeListedRequiresHosts(t *testing.T) {
	err := validateServiceMaintenance(
		service.Maintenance{Enabled: true, Scope: service.MaintenanceScopeListed},
		backendTypeService,
		"rules[0].backend.service",
		true,
	)
	if err == nil {
		t.Fatal("expected listed scope without hosts to fail")
	}
}

func TestValidateServiceMaintenance_ScopeAllForbidsHosts(t *testing.T) {
	err := validateServiceMaintenance(
		service.Maintenance{
			Enabled: true,
			Scope:   service.MaintenanceScopeAll,
			Hosts:   hosts(maintHost("app.example.com")),
		},
		backendTypeService,
		"rules[0].backend.service",
		true,
	)
	if err == nil {
		t.Fatal("expected scope all with hosts to fail")
	}
}

func TestMaintenanceBypassAllows_PathsAndHeader(t *testing.T) {
	b := compiledMaintenanceBypass{
		paths:   []maintenanceBypassPath{{exact: "/healthz"}},
		headers: []maintenanceBypassHeaderPair{{name: "X-Maintenance-Bypass", value: "secret"}},
	}
	req := &http.Request{Header: http.Header{}}
	if !maintenanceBypassAllows(b, req, "/healthz") {
		t.Fatal("expected path bypass")
	}
	req2 := &http.Request{Header: http.Header{"X-Maintenance-Bypass": {"secret"}}}
	if !maintenanceBypassAllows(b, req2, "/api") {
		t.Fatal("expected header bypass")
	}
}

func TestCompileMaintenanceByRule_ScopeListed(t *testing.T) {
	cfg := &Config{
		Rules: []rule.Rule{
			{
				Host: "*.example.com",
				Backend: rule.Backend{
					Type: backendTypeService,
					Service: service.Service{
						Name: "backend",
						Maintenance: service.Maintenance{
							Enabled: true,
							Scope:   service.MaintenanceScopeListed,
							Hosts:   hosts(maintHost("app.example.com")),
							Title:   "Maintenance",
						},
					},
				},
			},
		},
	}
	if err := inferBackendTypes(cfg); err != nil {
		t.Fatal(err)
	}
	if err := ValidateConfig(cfg); err != nil {
		t.Fatal(err)
	}
	out, err := compileMaintenanceByRule(cfg)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if !out[0].enabled || out[0].scopeAll || !out[0].hosts.MatchesActive("app.example.com", now) {
		t.Fatalf("unexpected compiled maintenance: %+v", out[0])
	}
}

func TestCompileGlobalMaintenance(t *testing.T) {
	g, err := compileGlobalMaintenance(MaintenanceConfig{
		Hosts: hosts(
			maintHost("app.example.com"),
			maintHost("staging-*.example.com"),
		),
		Title: "Global",
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if !g.hosts.MatchesActive("app.example.com", now) || !g.hosts.MatchesActive("staging-api.example.com", now) {
		t.Fatal("expected global host patterns to match")
	}
	if g.settings.Title != "Global" {
		t.Fatal("expected global title")
	}
}

func TestMaintenanceDecision_GlobalHost(t *testing.T) {
	c := &core{
		cfg: &Config{},
		globalMaintenance: mustCompileGlobal(t, MaintenanceConfig{
			Hosts: hosts(maintHost("app.example.com")),
			Title: "Global maintenance",
		}),
		maintenanceByRule: []compiledRuleMaintenance{},
	}
	block, settings, _ := c.maintenanceDecision(0, "app.example.com", "/api", &http.Request{Header: http.Header{}})
	if !block || settings.Title != "Global maintenance" {
		t.Fatalf("expected global maintenance, got block=%v settings=%+v", block, settings)
	}
}

func TestCompileMaintenanceWindow_InvalidEndBeforeStart(t *testing.T) {
	_, err := compileMaintenanceWindow(service.MaintenanceWindow{
		Start: "2026-05-30T06:00:00Z",
		End:   "2026-05-30T02:00:00Z",
	}, "maintenance.hosts[0].window")
	if err == nil {
		t.Fatal("expected end before start to fail")
	}
}

func TestCompiledMaintenanceWindow_ActiveAt(t *testing.T) {
	start := time.Date(2026, 5, 30, 2, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 30, 6, 0, 0, 0, time.UTC)
	w := compiledMaintenanceWindow{configured: true, start: start, end: end}

	if !w.activeAt(start) || !w.activeAt(end) || !w.activeAt(start.Add(2*time.Hour)) {
		t.Fatal("expected active inside window")
	}
	if w.activeAt(start.Add(-time.Minute)) || w.activeAt(end.Add(time.Minute)) {
		t.Fatal("expected inactive outside window")
	}
}

func TestMaintenanceDecision_RespectsPerHostWindow(t *testing.T) {
	c := &core{
		cfg: &Config{},
		globalMaintenance: mustCompileGlobal(t, MaintenanceConfig{
			Hosts: hosts(service.MaintenanceHostEntry{
				Host: "app.example.com",
				Window: service.MaintenanceWindow{
					Start: "2099-01-01T00:00:00Z",
					End:   "2099-01-02T00:00:00Z",
				},
			}),
			Title: "Global maintenance",
		}),
		maintenanceByRule: []compiledRuleMaintenance{},
	}
	block, _, _ := c.maintenanceDecision(0, "app.example.com", "/api", &http.Request{Header: http.Header{}})
	if block {
		t.Fatal("expected outside future per-host window to not block")
	}
}

func mustCompileGlobal(t *testing.T, cfg MaintenanceConfig) compiledGlobalMaintenance {
	t.Helper()
	g, err := compileGlobalMaintenance(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func mustCompileRuleMaintenance(t *testing.T, m service.Maintenance) compiledRuleMaintenance {
	t.Helper()
	out, err := compileServiceMaintenance(m, "rules[0].backend.service")
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestMaintenanceDecision_NonMatchingHostNoBlock(t *testing.T) {
	c := &core{
		cfg: &Config{},
		globalMaintenance: mustCompileGlobal(t, MaintenanceConfig{
			Hosts: hosts(maintHost("app.example.com")),
		}),
		maintenanceByRule: []compiledRuleMaintenance{},
	}
	block, _, _ := c.maintenanceDecision(0, "other.example.com", "/api", &http.Request{Header: http.Header{}})
	if block {
		t.Fatal("expected non-matching host to pass through")
	}
}

func TestMaintenanceDecision_RuleScopeAll(t *testing.T) {
	c := &core{
		cfg:               &Config{},
		globalMaintenance: compiledGlobalMaintenance{},
		maintenanceByRule: []compiledRuleMaintenance{
			mustCompileRuleMaintenance(t, service.Maintenance{
				Enabled: true,
				Scope:   service.MaintenanceScopeAll,
				Window:  maintRuleWindow(),
				Title:   "Rule maintenance",
			}),
		},
	}
	block, settings, _ := c.maintenanceDecision(0, "any.example.com", "/api", &http.Request{Header: http.Header{}})
	if !block || settings.Title != "Rule maintenance" {
		t.Fatalf("expected rule scope all block, got block=%v settings=%+v", block, settings)
	}
}

func TestMaintenanceDecision_RuleScopeListed(t *testing.T) {
	c := &core{
		cfg:               &Config{},
		globalMaintenance: compiledGlobalMaintenance{},
		maintenanceByRule: []compiledRuleMaintenance{
			mustCompileRuleMaintenance(t, service.Maintenance{
				Enabled: true,
				Scope:   service.MaintenanceScopeListed,
				Hosts:   hosts(maintHost("legacy.example.com")),
				Title:   "Legacy maintenance",
			}),
		},
	}
	block, settings, _ := c.maintenanceDecision(0, "legacy.example.com", "/api", &http.Request{Header: http.Header{}})
	if !block || settings.Title != "Legacy maintenance" {
		t.Fatalf("expected listed host block, got block=%v settings=%+v", block, settings)
	}
	block, _, _ = c.maintenanceDecision(0, "other.example.com", "/api", &http.Request{Header: http.Header{}})
	if block {
		t.Fatal("expected non-listed host to pass through")
	}
}

func TestMaintenanceDecision_MergeSettingsRuleOverridesGlobal(t *testing.T) {
	c := &core{
		cfg: &Config{},
		globalMaintenance: mustCompileGlobal(t, MaintenanceConfig{
			Hosts:      hosts(maintHost("app.example.com")),
			Title:      "Global title",
			RetryAfter: 60,
		}),
		maintenanceByRule: []compiledRuleMaintenance{
			mustCompileRuleMaintenance(t, service.Maintenance{
				Enabled:    true,
				Scope:      service.MaintenanceScopeAll,
				Window:     maintRuleWindow(),
				Title:      "Rule title",
				RetryAfter: 120,
			}),
		},
	}
	block, settings, _ := c.maintenanceDecision(0, "app.example.com", "/api", &http.Request{Header: http.Header{}})
	if !block {
		t.Fatal("expected block when global and rule both hit")
	}
	if settings.Title != "Rule title" || settings.RetryAfter != 120 {
		t.Fatalf("expected rule settings to override global, got %+v", settings)
	}
}

func TestMaintenanceDecision_GlobalOnlyUsesGlobalSettings(t *testing.T) {
	c := &core{
		cfg: &Config{},
		globalMaintenance: mustCompileGlobal(t, MaintenanceConfig{
			Hosts: hosts(maintHost("app.example.com")),
			Title: "Global title",
		}),
		maintenanceByRule: []compiledRuleMaintenance{
			mustCompileRuleMaintenance(t, service.Maintenance{
				Enabled: true,
				Scope:   service.MaintenanceScopeListed,
				Hosts:   hosts(maintHost("legacy.example.com")),
				Title:   "Rule title",
			}),
		},
	}
	block, settings, _ := c.maintenanceDecision(0, "app.example.com", "/api", &http.Request{Header: http.Header{}})
	if !block || settings.Title != "Global title" {
		t.Fatalf("expected global settings when only global hits, got block=%v settings=%+v", block, settings)
	}
}

func TestMaintenanceDecision_BypassPathInDecision(t *testing.T) {
	c := &core{
		cfg: &Config{},
		globalMaintenance: mustCompileGlobal(t, MaintenanceConfig{
			Hosts: hosts(maintHost("app.example.com")),
			Bypass: service.MaintenanceBypass{
				Paths: []string{"/healthz"},
			},
		}),
		maintenanceByRule: []compiledRuleMaintenance{},
	}
	block, _, _ := c.maintenanceDecision(0, "app.example.com", "/healthz", &http.Request{Header: http.Header{}})
	if block {
		t.Fatal("expected bypass path to skip maintenance block")
	}
	block, _, _ = c.maintenanceDecision(0, "app.example.com", "/api", &http.Request{Header: http.Header{}})
	if !block {
		t.Fatal("expected non-bypass path to block")
	}
}

func TestMaintenanceDecision_BypassHeaderInDecision(t *testing.T) {
	c := &core{
		cfg: &Config{},
		globalMaintenance: mustCompileGlobal(t, MaintenanceConfig{
			Hosts: hosts(maintHost("app.example.com")),
			Bypass: service.MaintenanceBypass{
				Header: service.MaintenanceBypassHeader{Name: "X-Maintenance-Bypass", Value: "secret"},
			},
		}),
		maintenanceByRule: []compiledRuleMaintenance{},
	}
	req := &http.Request{Header: http.Header{"X-Maintenance-Bypass": {"secret"}}}
	block, _, _ := c.maintenanceDecision(0, "app.example.com", "/api", req)
	if block {
		t.Fatal("expected bypass header to skip maintenance block")
	}
}

func TestMaintenanceBypassAllows_IPAllowlist(t *testing.T) {
	_, n, err := parseMaintenanceIPCIDR("10.0.0.0/24")
	if err != nil {
		t.Fatal(err)
	}
	b := compiledMaintenanceBypass{allowNets: []*net.IPNet{n}}
	req := httptest.NewRequest(http.MethodGet, "http://app.example.com/", nil)
	req.RemoteAddr = "10.0.0.42:12345"
	if !maintenanceBypassAllows(b, req, "/api") {
		t.Fatal("expected allowlisted IP to bypass")
	}
	req.RemoteAddr = "203.0.113.1:12345"
	if maintenanceBypassAllows(b, req, "/api") {
		t.Fatal("expected non-allowlisted IP to not bypass")
	}
}

func TestMaintenanceBypassAllows_PathPrefix(t *testing.T) {
	b := compiledMaintenanceBypass{
		paths: []maintenanceBypassPath{{prefix: "/healthz"}},
	}
	req := &http.Request{Header: http.Header{}}
	if !maintenanceBypassAllows(b, req, "/healthz/live") {
		t.Fatal("expected prefix bypass")
	}
	if maintenanceBypassAllows(b, req, "/api") {
		t.Fatal("expected non-matching path to not bypass")
	}
}

func TestCompiledMaintenanceWindow_OnlyStart(t *testing.T) {
	start := time.Date(2026, 5, 30, 2, 0, 0, 0, time.UTC)
	w := compiledMaintenanceWindow{configured: true, start: start}
	if !w.activeAt(start.Add(time.Hour)) {
		t.Fatal("expected active after start when end is open")
	}
	if w.activeAt(start.Add(-time.Minute)) {
		t.Fatal("expected inactive before start")
	}
}

func TestCompiledMaintenanceWindow_OnlyEnd(t *testing.T) {
	end := time.Date(2026, 5, 30, 6, 0, 0, 0, time.UTC)
	w := compiledMaintenanceWindow{configured: true, end: end}
	if !w.activeAt(end.Add(-time.Hour)) {
		t.Fatal("expected active before end when start is open")
	}
	if w.activeAt(end.Add(time.Minute)) {
		t.Fatal("expected inactive after end")
	}
}

func TestParseMaintenanceTime_InvalidRFC3339(t *testing.T) {
	_, err := parseMaintenanceTime("not-a-time", "maintenance.hosts[0].window.start")
	if err == nil {
		t.Fatal("expected invalid RFC3339 to fail")
	}
}

func TestCompileMaintenanceWindow_InvalidEndTimeRejected(t *testing.T) {
	_, err := compileMaintenanceWindow(service.MaintenanceWindow{
		Start: "2026-05-30T02:00:00Z",
		End:   "not-a-time",
	}, "maintenance.hosts[0].window")
	if err == nil {
		t.Fatal("expected invalid end time to fail")
	}
}

func TestCompileMaintenanceBypass_InvalidPathWildcard(t *testing.T) {
	_, err := compileMaintenanceBypass(service.MaintenanceBypass{
		Paths: []string{"/api/*/internal"},
	}, "maintenance.bypass")
	if err == nil {
		t.Fatal("expected mid-path wildcard to fail")
	}
}

func TestMaintenanceDecision_BypassMergesGlobalAndRule(t *testing.T) {
	c := &core{
		cfg: &Config{},
		globalMaintenance: mustCompileGlobal(t, MaintenanceConfig{
			Hosts: hosts(maintHost("app.example.com")),
			Bypass: service.MaintenanceBypass{
				Paths: []string{"/healthz"},
			},
		}),
		maintenanceByRule: []compiledRuleMaintenance{
			mustCompileRuleMaintenance(t, service.Maintenance{
				Enabled: true,
				Scope:   service.MaintenanceScopeAll,
				Window:  maintRuleWindow(),
				Bypass: service.MaintenanceBypass{
					Header: service.MaintenanceBypassHeader{Name: "X-Bypass", Value: "yes"},
				},
			}),
		},
	}
	req := &http.Request{Header: http.Header{"X-Bypass": {"yes"}}}
	block, _, _ := c.maintenanceDecision(0, "app.example.com", "/api", req)
	if block {
		t.Fatal("expected merged rule header bypass")
	}
	block, _, _ = c.maintenanceDecision(0, "app.example.com", "/healthz", &http.Request{Header: http.Header{}})
	if block {
		t.Fatal("expected merged global path bypass")
	}
}

func TestCompileIngressStatusPath_DefaultAndCustom(t *testing.T) {
	got, err := compileIngressStatusPath("")
	if err != nil || got != ingressStatusPathDefault {
		t.Fatalf("default: got=%q err=%v", got, err)
	}
	got, err = compileIngressStatusPath("/internal/ingress-status")
	if err != nil || got != "/internal/ingress-status" {
		t.Fatalf("custom: got=%q err=%v", got, err)
	}
	got, err = compileIngressStatusPath("internal/ingress-status")
	if err != nil || got != "/internal/ingress-status" {
		t.Fatalf("no leading slash: got=%q err=%v", got, err)
	}
	if _, err := compileIngressStatusPath("/../secret"); err == nil {
		t.Fatal("expected .. to fail")
	}
}

func TestValidateServiceMaintenance_ScopeAllRequiresWindow(t *testing.T) {
	err := validateServiceMaintenance(
		service.Maintenance{Enabled: true, Scope: service.MaintenanceScopeAll},
		backendTypeService,
		"rules[0].backend.service",
		true,
	)
	if err == nil || !strings.Contains(err.Error(), "maintenance.window") {
		t.Fatalf("expected scope all window error, got %v", err)
	}
}

func TestCompileMaintenanceHostList_RequiresWindow(t *testing.T) {
	_, err := compileMaintenanceHostList(service.MaintenanceHostList{
		{Host: "app.example.com"},
	}, "maintenance.hosts")
	if err == nil || !strings.Contains(err.Error(), "window") {
		t.Fatalf("expected window required, got %v", err)
	}
}

func TestCompileMaintenanceResponseHeader_Defaults(t *testing.T) {
	h, err := compileMaintenanceResponseHeader(service.MaintenanceResponseHeader{}, "maintenance.response_header")
	if err != nil {
		t.Fatal(err)
	}
	if h.name != headerXIngressMaintenance || h.value != ingressMaintenanceHeaderVal {
		t.Fatalf("unexpected defaults: %+v", h)
	}
}

func TestCompileMaintenanceResponseHeader_Custom(t *testing.T) {
	h, err := compileMaintenanceResponseHeader(service.MaintenanceResponseHeader{
		Name:  "X-Custom-Maintenance",
		Value: "yes",
	}, "maintenance.response_header")
	if err != nil {
		t.Fatal(err)
	}
	if h.name != "X-Custom-Maintenance" || h.value != "yes" {
		t.Fatalf("unexpected custom header: %+v", h)
	}
}

func TestCompileMaintenanceResponseHeader_PartialValueUsesDefaultName(t *testing.T) {
	h, err := compileMaintenanceResponseHeader(service.MaintenanceResponseHeader{
		Value: "on",
	}, "maintenance.response_header")
	if err != nil {
		t.Fatal(err)
	}
	if h.name != headerXIngressMaintenance || h.value != "on" {
		t.Fatalf("unexpected partial header: %+v", h)
	}
}

func TestMergeMaintenanceSettings_RuleResponseHeaderOverride(t *testing.T) {
	global := compiledMaintenanceSettings{
		responseHeader: compiledMaintenanceResponseHeader{
			name:  headerXIngressMaintenance,
			value: ingressMaintenanceHeaderVal,
		},
	}
	rule := compiledMaintenanceSettings{
		responseHeader: compiledMaintenanceResponseHeader{
			name:  "X-Custom-Maintenance",
			value: "yes",
		},
		responseHeaderConfigured: true,
	}
	merged := mergeMaintenanceSettings(global, rule, true)
	if merged.responseHeader.name != "X-Custom-Maintenance" || merged.responseHeader.value != "yes" {
		t.Fatalf("expected rule response header override, got %+v", merged.responseHeader)
	}
}

func TestPickEffectiveMaintenanceWindow_RuleOverridesGlobal(t *testing.T) {
	ruleWin := compiledMaintenanceWindow{configured: true, start: mustParseMaintTime(t, "2026-06-01T00:00:00Z"), end: mustParseMaintTime(t, "2026-06-01T06:00:00Z")}
	globalWin := compiledMaintenanceWindow{configured: true, end: mustParseMaintTime(t, "2026-12-31T23:59:59Z")}
	got := pickEffectiveMaintenanceWindow(true, ruleWin, true, globalWin)
	if !got.configured || !got.end.Equal(ruleWin.end) {
		t.Fatalf("expected rule window, got %+v", got)
	}
	got = pickEffectiveMaintenanceWindow(true, compiledMaintenanceWindow{}, true, globalWin)
	if !got.configured || !got.end.Equal(globalWin.end) {
		t.Fatalf("expected global window for scope:all rule, got %+v", got)
	}
}

func mustParseMaintTime(t *testing.T, raw string) time.Time {
	t.Helper()
	tm, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		t.Fatal(err)
	}
	return tm
}

func TestMaintenanceWindowHeaderValues(t *testing.T) {
	start := mustParseMaintTime(t, "2026-05-30T02:00:00+08:00")
	end := mustParseMaintTime(t, "2026-05-30T06:00:00+08:00")
	from, until := maintenanceWindowHeaderValues(compiledMaintenanceWindow{configured: true, start: start, end: end})
	if from != "2026-05-30T02:00:00+08:00" || until != "2026-05-30T06:00:00+08:00" {
		t.Fatalf("unexpected headers: from=%q until=%q", from, until)
	}
	from, until = maintenanceWindowHeaderValues(compiledMaintenanceWindow{})
	if from != "" || until != "" {
		t.Fatalf("expected empty for unconfigured window, got from=%q until=%q", from, until)
	}
}

func TestMaintenanceActiveForHost_Global(t *testing.T) {
	c := &core{
		cfg: &Config{},
		globalMaintenance: mustCompileGlobal(t, MaintenanceConfig{
			Hosts: hosts(maintHost("app.example.com")),
			Title: "Global maintenance",
		}),
		maintenanceByRule: []compiledRuleMaintenance{},
	}
	active, settings, _ := c.maintenanceActiveForHost(-1, "app.example.com", time.Now())
	if !active || settings.Title != "Global maintenance" {
		t.Fatalf("expected active global maintenance, got active=%v settings=%+v", active, settings)
	}
	active, _, _ = c.maintenanceActiveForHost(-1, "other.example.com", time.Now())
	if active {
		t.Fatal("expected inactive for non-listed host")
	}
}
