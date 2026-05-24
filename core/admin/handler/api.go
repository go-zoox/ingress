package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-zoox/ingress/core/admin/config"
	"github.com/go-zoox/ingress/core/admin/service"
	ingcore "github.com/go-zoox/ingress/core"
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
}

func NewAPI(cfg *config.Config) *API {
	logs := service.NewLogs(cfg)
	ingress := service.NewIngress(cfg)
	audit := service.NewAudit()
	return &API{
		cfg:      cfg,
		ingress:  ingress,
		logs:     logs,
		metrics:  service.NewMetrics(logs),
		audit:    audit,
		tls:      service.NewTLS(ingress),
		cache:    service.NewCache(ingress, logs),
		settings: service.NewSettings(cfg, ingress, logs),
		config:   service.NewConfig(ingress, audit),
	}
}

func (a *API) Mount(g *zoox.RouterGroup) {
	g.Get("/status", a.Status)
	g.Get("/routes", a.Routes)
	g.Post("/routes/match", a.Match)
	g.Post("/waf/toggle", a.WAFToggle)
	g.Get("/waf/events", a.WAFEvents)
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
	g.Post("/reload", a.Reload)
	g.Get("/logs", a.Logs)
	g.Get("/metrics/overview", a.OverviewMetrics)
	g.Get("/settings", a.Settings)
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
	ok(ctx, zoox.H{
		"version":            "ingress",
		"config_path":        a.ingress.ConfigPath(),
		"pid_file":           a.cfg.PidFile,
		"reload_ready":       a.ingress.ReloadReady(),
		"listen_http":        icfg.Port,
		"listen_https":       icfg.HTTPS.Port,
		"rules_count":        len(icfg.Rules),
		"waf_enabled":        wafOn,
		"waf_log_only":       icfg.WAF.LogOnly,
		"waf_runtime_enabled": wafRuntimeOn,
		"last_reload":        time.Now().Format(time.RFC3339),
		"config_hash":        a.configHash(),
	})
}

func (a *API) configHash() string {
	content, err := a.ingress.ReadYAML()
	if err != nil {
		return ""
	}
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:8])
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

func (a *API) WAFEvents(ctx *zoox.Context) {
	rows, err := a.audit.ListWAFEvents(100)
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ok(ctx, rows)
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
	q := service.LogQuery{
		Kind:     kind,
		Q:        strings.TrimSpace(ctx.Query().Get("q").String()),
		Host:     strings.TrimSpace(ctx.Query().Get("host").String()),
		Status:   strings.TrimSpace(ctx.Query().Get("status").String()),
		CacheHit: strings.TrimSpace(ctx.Query().Get("cache_hit").String()),
		WAFBlock: strings.TrimSpace(ctx.Query().Get("waf_block").String()),
		Limit:    limit,
		Offset:   offset,
	}
	result, err := a.logs.Search(q)
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ok(ctx, result)
}

func (a *API) OverviewMetrics(ctx *zoox.Context) {
	window := strings.TrimSpace(ctx.Query().Get("window").String())
	ok(ctx, a.metrics.Overview(window))
}

func (a *API) Settings(ctx *zoox.Context) {
	ok(ctx, a.settings.Get(a.configHash()))
}
