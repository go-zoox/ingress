package app

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/go-zoox/ingress/core/admin/bootstrap"
	"github.com/go-zoox/ingress/core/admin/config"
	"github.com/go-zoox/ingress/core/admin/handler"
	"github.com/go-zoox/ingress/core/admin/service"
	"github.com/go-zoox/ingress/core/admin/static"
	"github.com/go-zoox/logger"
	"github.com/go-zoox/zoox"
	"github.com/go-zoox/zoox/defaults"
)

// New builds the admin zoox application.
func New(cfg *config.Config) (*zoox.Application, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if err := bootstrap.Init(cfg); err != nil {
		return nil, err
	}

	// Wire WAF event callback so blocks/audits are persisted to the admin DB.
	if cfg.CoreInstance != nil {
		audit := service.NewAudit()
		cfg.CoreInstance.SetWAFCallback(&wafEventAdapter{audit: audit})
	}

	app := defaults.Default()
	app.SetBanner("")
	app.Config.Port = int(cfg.Port)

	api := handler.NewAPI(cfg)
	g := app.Group("/api/v1")
	api.Mount(g)

	// Start the health check service and SSE broker
	if api.Broker() != nil {
		api.Health().Start()
		// Note: Health check service will be stopped when the process exits.
		// zoox Application doesn't expose OnShutdown; cleanup is handled by process signals.
	}

	if !cfg.Web.DevProxy {
		if err := static.Mount(app); err != nil {
			return nil, fmt.Errorf("static: %w", err)
		}
	} else {
		app.Get("/", func(ctx *zoox.Context) {
			ctx.String(http.StatusOK, "ingress admin API (dev_proxy=true); run web with pnpm dev")
		})
	}

	return app, nil
}

// Run starts the admin HTTP server with an admin-specific startup log line.
func Run(app *zoox.Application, cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("admin config is nil")
	}
	if app.Config.NetworkType == "" {
		app.Config.NetworkType = "tcp"
	}
	if app.Config.Host == "" {
		app.Config.Host = "0.0.0.0"
	}
	app.Config.Port = int(cfg.Port)

	listener, err := net.Listen(app.Config.NetworkType, app.Address())
	if err != nil {
		return err
	}

	server := &http.Server{
		ReadTimeout:  300 * time.Second,
		WriteTimeout: 300 * time.Second,
		IdleTimeout:  300 * time.Second,
		Handler:      app,
	}

	app.Logger().Info("Admin started at http://%s", app.AddressForLog())
	return server.Serve(listener)
}

// wafEventAdapter bridges the core WAFCallback interface to the audit service.
type wafEventAdapter struct {
	audit *service.Audit
}

func (a *wafEventAdapter) OnWAFEvent(action, rule, host, path, clientIP string) {
	// Fire-and-forget to avoid blocking the request path.
	go func() {
		if err := a.audit.RecordWAFEvent(action, rule, host, path, clientIP); err != nil {
			logger.Warnf("waf event record failed: %s", err)
		}
	}()
}
