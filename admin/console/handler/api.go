package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-zoox/gormx"
	"github.com/go-zoox/ingress/admin/console/config"
	"github.com/go-zoox/ingress/admin/console/model"
	"github.com/go-zoox/ingress/admin/console/service"
	ingcore "github.com/go-zoox/ingress/core"
	"github.com/go-zoox/zoox"
)

// API bundles admin HTTP handlers.
type API struct {
	cfg     *config.Config
	ingress *service.Ingress
	logs    *service.Logs
	metrics *service.Metrics
	audit   *service.Audit
}

func NewAPI(cfg *config.Config) *API {
	logs := service.NewLogs(cfg)
	return &API{
		cfg:     cfg,
		ingress: service.NewIngress(cfg),
		logs:    logs,
		metrics: service.NewMetrics(logs),
		audit:   service.NewAudit(),
	}
}

func (a *API) Mount(g *zoox.RouterGroup) {
	g.Get("/status", a.Status)
	g.Get("/routes", a.Routes)
	g.Post("/routes/match", a.Match)
	g.Get("/waf/events", a.WAFEvents)
	g.Get("/tls/certs", a.TLSCerts)
	g.Get("/config", a.GetConfig)
	g.Put("/config", a.PutConfig)
	g.Post("/config/validate", a.ValidateConfig)
	g.Post("/reload", a.Reload)
	g.Get("/logs", a.Logs)
	g.Get("/metrics/overview", a.OverviewMetrics)
}

func (a *API) Status(ctx *zoox.Context) {
	icfg, err := a.ingress.LoadConfig()
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	wafOn := icfg.WAF.Enabled
	ok(ctx, zoox.H{
		"version":       "ingress",
		"config_path":   a.ingress.ConfigPath(),
		"pid_file":      a.cfg.Ingress.PidFile,
		"reload_ready":  a.ingress.ReloadReady(),
		"listen_http":   icfg.Port,
		"listen_https":  icfg.HTTPS.Port,
		"rules_count":   len(icfg.Rules),
		"waf_enabled":   wafOn,
		"waf_log_only":  icfg.WAF.LogOnly,
		"last_reload":   time.Now().Format(time.RFC3339),
		"config_hash":   a.configHash(),
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

func (a *API) WAFEvents(ctx *zoox.Context) {
	rows, err := a.audit.ListWAFEvents(100)
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ok(ctx, rows)
}

func (a *API) TLSCerts(ctx *zoox.Context) {
	icfg, err := a.ingress.LoadConfig()
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	type certRow struct {
		Domain   string `json:"domain"`
		Cert     string `json:"certificate"`
		Key      string `json:"certificate_key"`
		Status   string `json:"status"`
	}
	rows := make([]certRow, 0)
	for _, ssl := range icfg.HTTPS.SSL {
		st := "ok"
		if !fileExists(ssl.Cert.Certificate) {
			st = "missing"
		}
		rows = append(rows, certRow{
			Domain: ssl.Domain,
			Cert:   ssl.Cert.Certificate,
			Key:    ssl.Cert.CertificateKey,
			Status: st,
		})
	}
	ok(ctx, rows)
}

func fileExists(p string) bool {
	if p == "" {
		return false
	}
	_, err := os.Stat(p)
	return err == nil
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
	}
	if err := ctx.BindJSON(&body); err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if err := a.ingress.ValidateYAML(body.Content); err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if err := a.ingress.WriteYAML(body.Content); err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	sum := sha256.Sum256([]byte(body.Content))
	hash := hex.EncodeToString(sum[:8])
	_ = gormx.GetDB().Create(&model.ConfigRevision{
		Hash:      hash,
		Content:   body.Content,
		Note:      "save",
		CreatedAt: time.Now(),
	}).Error
	_ = a.audit.Record("config.save", hash, "admin")
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
	kind := service.LogAccess
	if strings.TrimSpace(ctx.Query().Get("log").String()) == "error" {
		kind = service.LogError
	}
	q := service.LogQuery{
		Kind:   kind,
		Q:      strings.TrimSpace(ctx.Query().Get("q").String()),
		Host:   strings.TrimSpace(ctx.Query().Get("host").String()),
		Status: strings.TrimSpace(ctx.Query().Get("status").String()),
		Limit:  limit,
	}
	lines, err := a.logs.Search(q)
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ok(ctx, zoox.H{"lines": lines, "count": len(lines)})
}

func (a *API) OverviewMetrics(ctx *zoox.Context) {
	window := strings.TrimSpace(ctx.Query().Get("window").String())
	ok(ctx, a.metrics.Overview(window))
}
