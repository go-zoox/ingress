package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-zoox/ingress"
	"github.com/go-zoox/ingress/core/admin/config"
	adminauth "github.com/go-zoox/ingress/core/admin/auth"
	"github.com/go-zoox/ingress/core/admin/service"
	ingcore "github.com/go-zoox/ingress/core"
	"github.com/go-zoox/ingress/core/admin/service/jobs"
	"github.com/go-zoox/ingress/core/admin/service/rbac"
	"github.com/go-zoox/zoox"
)

// API bundles admin HTTP handlers.
type API struct {
	cfg      *config.Config
	ingress  *service.Ingress
	logs     *service.Logs
	metrics  *service.Metrics
	audit    *service.Audit
	tls      *service.TLS
	cache    *service.Cache
	settings *service.Settings
	config   *service.Config
	scenarios *service.Scenarios
	broker   *service.SSEBroker
	health   *service.HealthCheckService
	system   *service.SystemMetrics
	parseIssues      *service.ParseIssues
	overviewBuilder  *service.OverviewBuilder
	overviewStreamer *service.OverviewStreamer
	jobs             *jobs.Service
	rbac             *rbac.Service
	auth             *adminauth.Service
}

func NewAPI(cfg *config.Config, auth *adminauth.Service) *API {
	logs := service.NewLogs(cfg)
	ingress := service.NewIngress(cfg)
	audit := service.NewAudit()
	broker := service.NewSSEBroker()
	healthSvc := service.NewHealthCheckService(ingress, broker)
	systemSvc := service.NewSystemMetrics()
	parseIssues := service.NewParseIssues()
	metrics := service.NewMetrics(logs, parseIssues)
	tlsSvc := service.NewTLS(ingress)
	configSvc := service.NewConfig(ingress, audit)
	scenariosSvc := service.NewScenarios(ingress, audit, configSvc)
	overviewBuilder := service.NewOverviewBuilder(ingress, metrics, systemSvc, tlsSvc, healthSvc, audit, parseIssues, configSvc)
	jobsSvc := jobs.New(cfg, ingress, audit, tlsSvc, metrics)
	return &API{
		cfg:              cfg,
		ingress:          ingress,
		logs:             logs,
		metrics:          metrics,
		audit:            audit,
		tls:              tlsSvc,
		cache:            service.NewCache(ingress, logs),
		settings:         service.NewSettings(cfg, ingress, logs),
		config:           configSvc,
		scenarios:        scenariosSvc,
		broker:           broker,
		health:           healthSvc,
		system:           systemSvc,
		parseIssues:      parseIssues,
		overviewBuilder:  overviewBuilder,
		overviewStreamer: service.NewOverviewStreamer(overviewBuilder, broker),
		jobs:             jobsSvc,
		rbac:             rbac.New(),
		auth:             auth,
	}
}

func (a *API) Mount(g *zoox.RouterGroup) {
	g.Get("/status", a.Status)
	g.Get("/routes", a.Routes)
	g.Get("/investigate", a.Investigate)
	g.Post("/routes/match", a.Match)
	g.Post("/waf/toggle", a.WAFToggle)
	g.Get("/waf/events", a.WAFEvents)
	g.Post("/waf/events/batch-status", a.BatchUpdateWAFEventStatus)
	g.Delete("/waf/events/demo-seed", a.ClearDemoWAFEvents)
	g.Get("/waf/events/:id", a.WAFEventDetail)
	g.Post("/waf/events/:id/status", a.UpdateWAFEventStatus)
	g.Post("/waf/match", a.WAFMatch)
	g.Get("/waf/visualization", a.WAFVisualization)
	g.Get("/waf/hosts", a.WAFHosts)
	g.Get("/waf/rules", a.WAFRules)
	g.Get("/waf/rules/catalog", a.WAFRulesCatalog)
	g.Get("/tls/certs", a.TLSCerts)
	g.Post("/tls/certs/check", a.TLSCertCheck)
	g.Get("/cache/overview", a.CacheOverview)
	g.Get("/config", a.GetConfig)
	g.Put("/config", a.PutConfig)
	g.Post("/config/validate", a.ValidateConfig)
	g.Post("/config/preview", a.PreviewConfig)
	g.Post("/config/publish", a.PublishConfig)
	g.Post("/config/modules", a.ConfigModules)
	g.Post("/config/modules/merge", a.MergeConfigModule)
	g.Get("/config/revisions", a.ConfigRevisions)
	g.Get("/config/revisions/:id", a.ConfigRevision)
	g.Get("/audit/logs", a.AuditLogs)
	g.Post("/reload", a.Reload)
	g.Get("/scenarios", a.Scenarios)
	g.Put("/scenarios/active", a.SetScenarioActive)
	g.Get("/logs", a.Logs)
	g.Get("/logs/hosts", a.LogHosts)
	g.Get("/metrics/overview", a.OverviewMetrics)
	g.Get("/metrics/system", a.SystemMetrics)
	g.Get("/overview/snapshot", a.OverviewSnapshot)
	g.Get("/logs/parse-issues", a.ListParseIssues)
	g.Post("/logs/parse-issues/batch-status", a.BatchUpdateParseIssueStatus)
	g.Get("/logs/parse-issues/:id", a.GetParseIssue)
	g.Post("/logs/parse-issues/:id/status", a.UpdateParseIssueStatus)
	g.Get("/settings", a.Settings)
	// New routes for SSE, route detail, and health check
	sseHandler := NewSSEHandler(a.broker, a.overviewStreamer)
	g.Get("/events/stream", sseHandler.Stream)
	routeDetailHandler := NewRouteDetailHandler(a.ingress, a.metrics, a.health, a.audit)
	g.Get("/routes/:ri/:pi", routeDetailHandler.GetDetail)
	g.Get("/routes/:ri/:pi/metrics", routeDetailHandler.GetMetrics)
	serviceDetailHandler := NewServiceDetailHandler(a.ingress, a.health)
	g.Get("/services/:name", serviceDetailHandler.GetDetail)
	g.Get("/services/:name/metrics", serviceDetailHandler.GetMetrics)
	healthHandler := NewHealthHandler(a.health)
	g.Get("/healthcheck", healthHandler.ListChecks)
	jobsHandler := NewJobsHandler(a.jobs)
	jobsHandler.Mount(g)
	rbacHandler := NewRBACHandler(a.rbac, a.auth)
	rbacHandler.Mount(g)
	if err := MountTerminal(g); err != nil {
		panic(err)
	}
}

func (a *API) Status(ctx *zoox.Context) {
	icfg, err := a.ingress.LoadConfig()
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	wafOn := icfg.WAF.Enabled
	wafRuntimeOn := wafOn
	if a.cfg.CoreInstance != nil {
		wafRuntimeOn = a.cfg.CoreInstance.IsWAFEnabled()
	}
	fileHash := a.fileConfigHash()
	runtimeHash := a.runtimeConfigHash()
	revs, _ := a.config.ListRevisions(1)
	latestRevisionHash := ""
	if len(revs) > 0 {
		latestRevisionHash = revs[0].Hash
	}
	ok(ctx, zoox.H{
		"version":              ingress.Version,
		"config_path":          a.ingress.ConfigPath(),
		"pid_file":             a.cfg.PidFile,
		"reload_ready":         a.ingress.ReloadReady(),
		"listen_http":          icfg.Port,
		"listen_https":         icfg.HTTPS.Port,
		"rules_count":          len(icfg.Rules),
		"waf_enabled":          wafOn,
		"waf_log_only":         icfg.WAF.LogOnly,
		"waf_runtime_enabled":  wafRuntimeOn,
		"last_reload":          time.Now().Format(time.RFC3339),
		"config_hash":          fileHash,
		"file_hash":            fileHash,
		"runtime_hash":         runtimeHash,
		"latest_revision_hash": latestRevisionHash,
		"runtime_drift":        runtimeHash != "" && fileHash != "" && runtimeHash != fileHash,
		"revision_drift":       latestRevisionHash != "" && fileHash != "" && fileHash != latestRevisionHash,
	})
}

func (a *API) fileConfigHash() string {
	content, err := a.ingress.ReadYAML()
	if err != nil {
		return ""
	}
	return ingcore.ContentHash(content)
}

func (a *API) runtimeConfigHash() string {
	if a.cfg.CoreInstance == nil {
		return ""
	}
	return a.cfg.CoreInstance.ConfigFingerprint()
}

func (a *API) configHash() string {
	return a.fileConfigHash()
}

func (a *API) AuditLogs(ctx *zoox.Context) {
	limit, _ := strconv.Atoi(strings.TrimSpace(ctx.Query().Get("limit").String()))
	rows, err := a.audit.List(limit)
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	if rows == nil {
		rows = []service.AuditLogRow{}
	}
	ok(ctx, rows)
}

func (a *API) Routes(ctx *zoox.Context) {
	icfg, err := a.ingress.LoadConfig()
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	list, err := ingcore.ListRouteRows(icfg)
	if err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if list == nil {
		list = []ingcore.RouteRow{}
	}
	ok(ctx, list)
}

func (a *API) Match(ctx *zoox.Context) {
	var body struct {
		Host string `json:"host"`
		Path string `json:"path"`
	}
	if err := ctx.BindJSON(&body); err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	icfg, err := a.ingress.LoadConfig()
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	preview, err := ingcore.PreviewMatch(icfg, body.Host, body.Path)
	if err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ok(ctx, preview)
}

func (a *API) WAFToggle(ctx *zoox.Context) {
	var body struct {
		Enabled *bool `json:"enabled"`
	}
	if err := ctx.BindJSON(&body); err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if a.cfg.CoreInstance == nil {
		fail(ctx, http.StatusServiceUnavailable, "core not available")
		return
	}
	a.cfg.CoreInstance.SetWAFOverride(body.Enabled)
	ok(ctx, zoox.H{"ok": true})
}

func (a *API) ClearDemoWAFEvents(ctx *zoox.Context) {
	n, err := a.audit.ClearDemoWAFEvents()
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ok(ctx, zoox.H{"ok": true, "deleted": n})
}

func (a *API) WAFEvents(ctx *zoox.Context) {
	f := service.WAFAuditFilter{}

	pathMatch := strings.ToLower(strings.TrimSpace(ctx.Query().Get("path_match").String()))
	if pathMatch == "" {
		pathMatch = "prefix"
	}

	// query params
	if v := strings.TrimSpace(ctx.Query().Get("action").String()); v != "" {
		f.Action = v
	}
	if v := strings.TrimSpace(ctx.Query().Get("host").String()); v != "" {
		f.Host = v
	}
	if v := strings.TrimSpace(ctx.Query().Get("path").String()); v != "" {
		f.Path = v
	}
	if v := strings.TrimSpace(ctx.Query().Get("client_ip").String()); v != "" {
		f.ClientIP = v
	}
	if v := strings.TrimSpace(ctx.Query().Get("rule").String()); v != "" {
		f.Rule = v
	}
	if v := strings.TrimSpace(ctx.Query().Get("status").String()); v != "" {
		f.Status = v
	}
	if v := strings.TrimSpace(ctx.Query().Get("time_start").String()); v != "" {
		f.TimeStart = v
	}
	if v := strings.TrimSpace(ctx.Query().Get("time_end").String()); v != "" {
		f.TimeEnd = v
	}
	if v := strings.TrimSpace(ctx.Query().Get("limit").String()); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			f.Limit = n
		}
	}
	scopeHostOverride := strings.TrimSpace(f.Host)
	scopePathOverride := strings.TrimSpace(f.Path)
	if _, _, routeFilter := parseRouteQueryIndices(ctx); routeFilter {
		if f.Limit < 500 {
			f.Limit = 500
		}
		// For prefix matching we don't want DB exact-path filtering to hide matches.
		if scopePathOverride != "" && pathMatch == "prefix" {
			f.Path = ""
		}
	}

	rows, err := a.audit.ListWAFEvents(f)
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	if ri, pi, ok := parseRouteQueryIndices(ctx); ok {
		if icfg, err := a.ingress.LoadConfig(); err == nil {
			rows = service.FilterWAFEventsForRoute(icfg, ri, pi, rows)
		}
	}

	// Optional scope overrides (used by route detail page).
	if scopeHostOverride != "" || scopePathOverride != "" {
		scoped := rows[:0]
		for _, row := range rows {
			if scopeHostOverride != "" && !strings.EqualFold(strings.TrimSpace(row.Host), scopeHostOverride) {
				continue
			}
			if scopePathOverride != "" && !service.MatchPathForScope(row.Path, scopePathOverride, pathMatch) {
				continue
			}
			scoped = append(scoped, row)
		}
		rows = scoped
	}
	ok(ctx, rows)
}

func (a *API) WAFVisualization(ctx *zoox.Context) {
	f := service.WAFAuditFilter{Limit: 500}

	pathMatch := strings.ToLower(strings.TrimSpace(ctx.Query().Get("path_match").String()))
	if pathMatch == "" {
		pathMatch = "prefix"
	}

	if v := strings.TrimSpace(ctx.Query().Get("action").String()); v != "" {
		f.Action = v
	}
	if v := strings.TrimSpace(ctx.Query().Get("host").String()); v != "" {
		f.Host = v
	}
	if v := strings.TrimSpace(ctx.Query().Get("path").String()); v != "" {
		f.Path = v
	}
	if v := strings.TrimSpace(ctx.Query().Get("client_ip").String()); v != "" {
		f.ClientIP = v
	}
	if v := strings.TrimSpace(ctx.Query().Get("rule").String()); v != "" {
		f.Rule = v
	}
	if v := strings.TrimSpace(ctx.Query().Get("status").String()); v != "" {
		f.Status = v
	}
	if v := strings.TrimSpace(ctx.Query().Get("time_start").String()); v != "" {
		f.TimeStart = v
	}
	if v := strings.TrimSpace(ctx.Query().Get("time_end").String()); v != "" {
		f.TimeEnd = v
	}
	scopeHostOverride := strings.TrimSpace(f.Host)
	scopePathOverride := strings.TrimSpace(f.Path)
	if _, _, routeFilter := parseRouteQueryIndices(ctx); routeFilter {
		if scopePathOverride != "" && pathMatch == "prefix" {
			f.Path = ""
		}
	}

	rows, err := a.audit.ListWAFEvents(f)
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	if ri, pi, ok := parseRouteQueryIndices(ctx); ok {
		if icfg, err := a.ingress.LoadConfig(); err == nil {
			rows = service.FilterWAFEventsForRoute(icfg, ri, pi, rows)
		}
	}
	if scopeHostOverride != "" || scopePathOverride != "" {
		scoped := rows[:0]
		for _, row := range rows {
			if scopeHostOverride != "" && !strings.EqualFold(strings.TrimSpace(row.Host), scopeHostOverride) {
				continue
			}
			if scopePathOverride != "" && !service.MatchPathForScope(row.Path, scopePathOverride, pathMatch) {
				continue
			}
			scoped = append(scoped, row)
		}
		rows = scoped
	}

	ok(ctx, service.BuildWAFVisualization(rows))
}

func (a *API) WAFEventDetail(ctx *zoox.Context) {
	id, err := strconv.ParseUint(strings.TrimSpace(ctx.Param().Get("id").String()), 10, 64)
	if err != nil || id == 0 {
		fail(ctx, http.StatusBadRequest, "invalid event id")
		return
	}
	row, err := a.audit.GetWAFEvent(uint(id))
	if err != nil {
		fail(ctx, http.StatusNotFound, "event not found")
		return
	}
	cfg, _ := a.ingress.LoadConfig()
	detail := service.BuildWAFEventDetail(cfg, row)
	ok(ctx, detail)
}

func (a *API) UpdateWAFEventStatus(ctx *zoox.Context) {
	id, err := strconv.ParseUint(strings.TrimSpace(ctx.Param().Get("id").String()), 10, 64)
	if err != nil || id == 0 {
		fail(ctx, http.StatusBadRequest, "invalid event id")
		return
	}
	var body struct {
		Status string `json:"status"`
		Note   string `json:"note"`
	}
	if err := ctx.BindJSON(&body); err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	row, err := a.audit.SetWAFEventStatus(uint(id), body.Status, body.Note)
	if err != nil {
		fail(ctx, http.StatusNotFound, err.Error())
		return
	}
	ok(ctx, row)
}

func (a *API) BatchUpdateWAFEventStatus(ctx *zoox.Context) {
	var body struct {
		IDs    []uint `json:"ids"`
		Status string `json:"status"`
		Note   string `json:"note"`
	}
	if err := ctx.BindJSON(&body); err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if len(body.IDs) == 0 {
		fail(ctx, http.StatusBadRequest, "ids required")
		return
	}
	n, err := a.audit.BatchSetWAFEventStatus(body.IDs, body.Status, body.Note)
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ok(ctx, map[string]any{"ok": true, "updated": n})
}

func (a *API) WAFMatch(ctx *zoox.Context) {
	var body service.WAFTrialInput
	if err := ctx.BindJSON(&body); err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	out, err := a.ingress.TrialWAF(body)
	if err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ok(ctx, out)
}

func (a *API) WAFHosts(ctx *zoox.Context) {
	hosts, err := a.audit.DistinctWAFHosts()
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	if hosts == nil {
		hosts = []string{}
	}
	ok(ctx, hosts)
}

func (a *API) WAFRules(ctx *zoox.Context) {
	rules, err := a.audit.DistinctWAFRules()
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	if rules == nil {
		rules = []string{}
	}
	ok(ctx, rules)
}

func (a *API) WAFRulesCatalog(ctx *zoox.Context) {
	cfg, err := a.ingress.LoadConfig()
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	catalog := service.WAFRulesCatalog(cfg)
	if catalog == nil {
		catalog = []service.WAFRuleDetail{}
	}
	ok(ctx, catalog)
}

func (a *API) TLSCerts(ctx *zoox.Context) {
	rows, err := a.tls.List()
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	if rows == nil {
		rows = []service.TLSCertRow{}
	}
	ok(ctx, rows)
}

func (a *API) TLSCertCheck(ctx *zoox.Context) {
	var body struct {
		Domain string `json:"domain"`
	}
	if err := ctx.BindJSON(&body); err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	out, err := a.tls.Inspect(body.Domain)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			fail(ctx, http.StatusNotFound, err.Error())
			return
		}
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ok(ctx, out)
}

func (a *API) CacheOverview(ctx *zoox.Context) {
	out, err := a.cache.Overview()
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ok(ctx, out)
}

func (a *API) GetConfig(ctx *zoox.Context) {
	content, err := a.ingress.ReadYAML()
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ok(ctx, zoox.H{"path": a.ingress.ConfigPath(), "content": content})
}

func (a *API) PutConfig(ctx *zoox.Context) {
	var body struct {
		Content string `json:"content"`
		Note    string `json:"note"`
	}
	if err := ctx.BindJSON(&body); err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	hash, err := a.config.Save(body.Content, body.Note)
	if err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ok(ctx, zoox.H{"hash": hash})
}

func (a *API) ValidateConfig(ctx *zoox.Context) {
	var body struct {
		Content string `json:"content"`
	}
	if err := ctx.BindJSON(&body); err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	content := body.Content
	if content == "" {
		var err error
		content, err = a.ingress.ReadYAML()
		if err != nil {
			fail(ctx, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if err := a.ingress.ValidateYAML(content); err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ok(ctx, zoox.H{"valid": true})
}

func (a *API) PreviewConfig(ctx *zoox.Context) {
	var body struct {
		Content string `json:"content"`
	}
	if err := ctx.BindJSON(&body); err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	out, err := a.config.Preview(body.Content)
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ok(ctx, out)
}

func (a *API) PublishConfig(ctx *zoox.Context) {
	var body struct {
		Content string `json:"content"`
		Note    string `json:"note"`
	}
	if err := ctx.BindJSON(&body); err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	hash, err := a.config.Publish(body.Content, body.Note)
	if err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ok(ctx, zoox.H{"hash": hash, "ok": true})
}

func (a *API) ConfigModules(ctx *zoox.Context) {
	var body struct {
		Content string `json:"content"`
	}
	if err := ctx.BindJSON(&body); err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	modules, err := a.config.Modules(body.Content)
	if err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ok(ctx, modules)
}

func (a *API) MergeConfigModule(ctx *zoox.Context) {
	var body struct {
		Content    string `json:"content"`
		ModuleID   string `json:"module_id"`
		ModuleYAML string `json:"module_yaml"`
	}
	if err := ctx.BindJSON(&body); err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	out, err := a.config.ApplyModule(body.Content, body.ModuleID, body.ModuleYAML)
	if err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ok(ctx, zoox.H{"content": out})
}

func (a *API) ConfigRevisions(ctx *zoox.Context) {
	limit := 50
	if v := strings.TrimSpace(ctx.Query().Get("limit").String()); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	rows, err := a.config.ListRevisions(limit)
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	if rows == nil {
		rows = []service.ConfigRevisionSummary{}
	}
	ok(ctx, rows)
}

func (a *API) ConfigRevision(ctx *zoox.Context) {
	id, err := strconv.ParseUint(strings.TrimSpace(ctx.Param().Get("id").String()), 10, 64)
	if err != nil || id == 0 {
		fail(ctx, http.StatusBadRequest, "invalid revision id")
		return
	}
	row, err := a.config.GetRevision(uint(id))
	if err != nil {
		fail(ctx, http.StatusNotFound, err.Error())
		return
	}
	ok(ctx, row)
}

func (a *API) Reload(ctx *zoox.Context) {
	if err := a.ingress.Reload(); err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	service.SyncGeoIPFromIngress(a.ingress)
	if a.jobs != nil {
		_ = a.jobs.Reload()
	}
	_ = a.audit.Record("ingress.reload", a.ingress.ConfigPath(), "admin")
	ok(ctx, zoox.H{"ok": true})
}

func (a *API) Logs(ctx *zoox.Context) {
	limit := 200
	if v := strings.TrimSpace(ctx.Query().Get("limit").String()); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	var offset int64
	if v := strings.TrimSpace(ctx.Query().Get("offset").String()); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n >= 0 {
			offset = n
		}
	}
	kind := service.LogAccess
	if strings.TrimSpace(ctx.Query().Get("log").String()) == "error" {
		kind = service.LogError
	}
	pathMatch := strings.ToLower(strings.TrimSpace(ctx.Query().Get("path_match").String()))
	if pathMatch == "" {
		pathMatch = "prefix"
	}
	q := service.LogQuery{
		Kind:     kind,
		Q:        strings.TrimSpace(ctx.Query().Get("q").String()),
		Host:     strings.TrimSpace(ctx.Query().Get("host").String()),
		Path:     strings.TrimSpace(ctx.Query().Get("path").String()),
		Status:   strings.TrimSpace(ctx.Query().Get("status").String()),
		CacheHit: strings.TrimSpace(ctx.Query().Get("cache_hit").String()),
		WAFBlock: strings.TrimSpace(ctx.Query().Get("waf_block").String()),
		Limit:    limit,
		Offset:   offset,
	}
	if ri, pi, routeFilter := parseRouteQueryIndices(ctx); routeFilter && q.Offset == 0 {
		icfg, err := a.ingress.LoadConfig()
		if err != nil {
			fail(ctx, http.StatusInternalServerError, err.Error())
			return
		}
		lines, err := a.logs.TailAccess(5000)
		if err != nil {
			fail(ctx, http.StatusInternalServerError, err.Error())
			return
		}
		filtered := service.FilterAccessLinesForRoute(icfg, ri, pi, lines)
		// Optional exact scope overrides: used by route detail page.
		if q.Host != "" || q.Path != "" {
			scoped := make([]string, 0, len(filtered))
			for _, line := range filtered {
				e, ok := service.ParseAccessEntry(line)
				if !ok {
					continue
				}
				if q.Host != "" && !strings.EqualFold(strings.TrimSpace(e.Host), q.Host) {
					continue
				}
				if q.Path != "" && !service.MatchPathForScope(e.Path, q.Path, pathMatch) {
					continue
				}
				scoped = append(scoped, line)
			}
			filtered = scoped
		}
		if q.Limit > 0 && len(filtered) > q.Limit {
			filtered = filtered[len(filtered)-q.Limit:]
		}
		var logOffset int64
		if path := a.logs.AccessLogPath(); path != "" {
			if size, err := service.LogFileSize(path); err == nil {
				logOffset = size
			}
		}
		ok(ctx, service.LogResult{Lines: filtered, Count: len(filtered), Offset: logOffset})
		return
	}

	result, err := a.logs.Search(q)
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ok(ctx, result)
}

func (a *API) LogHosts(ctx *zoox.Context) {
	hosts, err := a.logs.DistinctHosts()
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	if hosts == nil {
		hosts = []string{}
	}
	ok(ctx, hosts)
}

func (a *API) OverviewMetrics(ctx *zoox.Context) {
	q, err := parseMetricsRangeQuery(ctx)
	if err != nil {
		return
	}
	ok(ctx, a.metrics.OverviewWithRange(q))
}

func (a *API) OverviewSnapshot(ctx *zoox.Context) {
	q, err := parseMetricsRangeQuery(ctx)
	if err != nil {
		return
	}
	ok(ctx, a.overviewBuilder.SnapshotWithRange(q))
}

func (a *API) SystemMetrics(ctx *zoox.Context) {
	q, err := parseMetricsRangeQuery(ctx)
	if err != nil {
		return
	}
	ok(ctx, a.system.SnapshotWithRange(q))
}

func parseMetricsRangeQuery(ctx *zoox.Context) (service.MetricsRangeQuery, error) {
	fromStr := strings.TrimSpace(ctx.Query().Get("from").String())
	toStr := strings.TrimSpace(ctx.Query().Get("to").String())
	if fromStr != "" && toStr != "" {
		q, err := service.ParseMetricsRangeQuery(fromStr, toStr)
		if err != nil {
			fail(ctx, http.StatusBadRequest, err.Error())
			return q, err
		}
		return q, nil
	}
	// Legacy: ?window=5m until clients migrate.
	if window := strings.TrimSpace(ctx.Query().Get("window").String()); window != "" {
		return service.MetricsRangeFromWindow(window), nil
	}
	fail(ctx, http.StatusBadRequest, "from and to are required (RFC3339)")
	return service.MetricsRangeQuery{}, fmt.Errorf("from and to required")
}

func (a *API) ListParseIssues(ctx *zoox.Context) {
	status := strings.TrimSpace(ctx.Query().Get("status").String())
	if status == "" {
		status = "open"
	}
	limit := 20
	if v := strings.TrimSpace(ctx.Query().Get("limit").String()); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	rows, err := a.parseIssues.List(status, limit)
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	if rows == nil {
		rows = []service.AccessLogParseIssueRow{}
	}
	ok(ctx, rows)
}

func (a *API) GetParseIssue(ctx *zoox.Context) {
	id, err := strconv.ParseUint(strings.TrimSpace(ctx.Param().Get("id").String()), 10, 64)
	if err != nil || id == 0 {
		fail(ctx, http.StatusBadRequest, "invalid id")
		return
	}
	row, err := a.parseIssues.GetDetail(uint(id), a.logs)
	if err != nil {
		fail(ctx, http.StatusNotFound, err.Error())
		return
	}
	ok(ctx, row)
}

func (a *API) UpdateParseIssueStatus(ctx *zoox.Context) {
	id, err := strconv.ParseUint(strings.TrimSpace(ctx.Param().Get("id").String()), 10, 64)
	if err != nil || id == 0 {
		fail(ctx, http.StatusBadRequest, "invalid id")
		return
	}
	var body struct {
		Status string `json:"status"`
		Note   string `json:"note"`
	}
	if err := ctx.BindJSON(&body); err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	row, err := a.parseIssues.SetStatus(uint(id), body.Status, body.Note)
	if err != nil {
		fail(ctx, http.StatusNotFound, err.Error())
		return
	}
	ok(ctx, row)
}

func (a *API) BatchUpdateParseIssueStatus(ctx *zoox.Context) {
	var body struct {
		IDs    []uint `json:"ids"`
		Status string `json:"status"`
		Note   string `json:"note"`
	}
	if err := ctx.BindJSON(&body); err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if len(body.IDs) == 0 {
		fail(ctx, http.StatusBadRequest, "ids required")
		return
	}
	n, err := a.parseIssues.BatchSetStatus(body.IDs, body.Status, body.Note)
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ok(ctx, map[string]any{"ok": true, "updated": n})
}

func (a *API) Settings(ctx *zoox.Context) {
	ok(ctx, a.settings.Get(a.configHash()))
}

// Broker returns the SSE broker instance.
func (a *API) Broker() *service.SSEBroker {
	return a.broker
}

func (a *API) Jobs() *jobs.Service {
	return a.jobs
}

// LogsService returns the logs service instance.
func (a *API) LogsService() *service.Logs {
	return a.logs
}

// MetricsService returns the metrics aggregator.
func (a *API) MetricsService() *service.Metrics {
	return a.metrics
}

// Health returns the health check service instance.
func (a *API) Health() *service.HealthCheckService {
	return a.health
}

// SystemMetricsService returns the process metrics sampler.
func (a *API) SystemMetricsService() *service.SystemMetrics {
	return a.system
}

// OverviewStreamer returns the overview SSE publisher.
func (a *API) OverviewStreamer() *service.OverviewStreamer {
	return a.overviewStreamer
}
