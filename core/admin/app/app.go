package app

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/go-zoox/ingress/core/admin/bootstrap"
	"github.com/go-zoox/ingress/core/admin/config"
	adminauth "github.com/go-zoox/ingress/core/admin/auth"
	"github.com/go-zoox/ingress/core/admin/handler"
	"github.com/go-zoox/ingress/core/admin/service"
	"github.com/go-zoox/ingress/core/admin/service/rbac"
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

	app := defaults.Default()
	app.SetBanner("")
	applyServeDefaults(app)
	app.Config.Port = int(cfg.Port)

	rbacSvc := rbac.New()
	authSvc := adminauth.New(cfg, rbacSvc)
	api := handler.NewAPI(cfg, authSvc)

	g := app.Group("/api/v1")
	g.Use(authSvc.Middleware())
	authHandler := handler.NewAuthHandler(authSvc)
	authHandler.Mount(g)
	api.Mount(g)

	// Wire WAF event callback so blocks/audits are persisted and pushed over SSE.
	if cfg.CoreInstance != nil {
		audit := service.NewAudit()
		cfg.CoreInstance.SetWAFCallback(&wafEventAdapter{audit: audit, broker: api.Broker()})
	}

	// Start the health check service, log tail SSE, and SSE broker
	if api.Broker() != nil {
		logStreamer := service.NewLogStreamer(api.LogsService(), api.Broker())
		logStreamer.SetOnAccessLine(func() {
			api.OverviewStreamer().PushAll()
		})
		logStreamer.Start(2 * time.Second)
		api.OverviewStreamer().Start(5 * time.Second)
		api.Health().Start()
		api.SystemMetricsService().Start()
		if err := api.Jobs().Start(app.Cron()); err != nil {
			logger.Warnf("jobs scheduler: %v", err)
		}
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
	applyServeDefaults(app)
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

// applyServeDefaults mirrors zoox Run() defaults required when serving via http.Server directly.
// Without a stable SecretKey, session cookies cannot be decrypted on subsequent requests.
func applyServeDefaults(app *zoox.Application) {
	if app == nil {
		return
	}
	if app.Config.SecretKey == "" {
		if v := os.Getenv(zoox.BuiltInEnvSecretKey); v != "" {
			app.Config.SecretKey = v
		} else {
			app.Config.SecretKey = zoox.DefaultSecretKey
		}
	}
	if app.Config.Session.MaxAge == 0 {
		app.Config.Session.MaxAge = zoox.DefaultSessionMaxAge
	}
}

// wafEventAdapter bridges the core WAFCallback interface to the audit service.
type wafEventAdapter struct {
	audit  *service.Audit
	broker *service.SSEBroker
}

func (a *wafEventAdapter) OnWAFEvent(action, rule, host, path, clientIP, userAgent string) {
	// Fire-and-forget to avoid blocking the request path.
	go func() {
		row, err := a.audit.RecordWAFEvent(action, rule, host, path, clientIP, userAgent)
		if err != nil {
			logger.Warnf("waf event record failed: %s", err)
			return
		}
		if a.broker != nil && row != nil {
			a.broker.PublishJSON("waf", "event", row)
		}
	}()
}
