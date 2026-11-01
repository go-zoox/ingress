package core

import (
	"os"
	"sync"

	"github.com/go-zoox/debug"
	"github.com/go-zoox/ingress/core/peerip"
	"github.com/go-zoox/zoox"
	"github.com/go-zoox/zoox/middleware"
)

var appBuildMu sync.Mutex

// newIngressApplication builds the zoox application with the same middleware
// stack as zoox defaults.Default(), except that the real TCP peer is captured
// before the realip middleware rewrites RemoteAddr from client-controlled
// forwarding headers (True-Client-IP / X-Real-IP / X-Forwarded-For).
//
// WAF IP allowlists must not trust RemoteAddr alone, so the captured peer is
// exposed through core/peerip for security decisions.
//
// Keep the middleware and before-ready hook list in sync with zoox
// defaults.Default() when upgrading zoox.
func newIngressApplication() *zoox.Application {
	// defaults.Default() performs global Sentry init before creating the app.
	if os.Getenv(zoox.BuiltInEnvMonitorSentryEnabled) == "true" && os.Getenv(zoox.BuiltInEnvMonitorSentryDSN) != "" {
		middleware.InitSentry(middleware.InitSentryOption{
			Dsn:   os.Getenv(zoox.BuiltInEnvMonitorSentryDSN),
			Debug: debug.IsDebugMode(),
		})
	}

	// zoox.New() applies every registered default middleware factory. Swap the
	// global registry for the duration of app construction so the ingress app
	// gets exactly this ordered stack (no duplicate defaults).
	appBuildMu.Lock()
	saved := zoox.DefaultMiddlewares
	zoox.DefaultMiddlewares = map[string]func(*zoox.Application){
		"ingress": func(app *zoox.Application) {
			app.Use(middleware.Recovery())
			app.Use(middleware.RequestID())
			app.Use(peerIPCaptureMiddleware())
			app.Use(middleware.RealIP())
			app.Use(func(ctx *zoox.Context) {
				if ctx.App.Config.Logger.Middleware.Disabled {
					ctx.Next()
					return
				}
				middleware.Logger()(ctx)
			})
			app.Use(middleware.HealthCheck())
			app.Use(middleware.Runtime())
			if debug.IsDebugMode() {
				app.Use(middleware.PProf())
			}
		},
	}
	app := zoox.New()
	zoox.DefaultMiddlewares = saved
	appBuildMu.Unlock()

	// Re-register the before-ready hooks from defaults.Default().
	app.SetBeforeReady(func() {
		if app.Config.BodySizeLimit > 0 {
			app.Logger().Infof("[middleware] register: body limit (app.Config) ...")
			app.Use(middleware.BodyLimit(func(cfg *middleware.BodyLimitConfig) {
				cfg.MaxSize = app.Config.BodySizeLimit
			}))
		}

		if app.Config.Monitor.Prometheus.Enabled {
			app.Logger().Infof("[middleware] register: prometheus (app.Config) ...")
			app.Use(middleware.Prometheus(func(opt *middleware.PrometheusOption) {
				if app.Config.Monitor.Prometheus.Path != "" {
					opt.Path = app.Config.Monitor.Prometheus.Path
				}
			}))
		}

		if app.Config.Monitor.Sentry.Enabled {
			if app.Config.Monitor.Sentry.DSN == "" {
				panic("app.Config.Monitor.Sentry.DSN is required")
			}
			app.Logger().Infof("[middleware] register: sentry (app.Config) ...")
			if os.Getenv(zoox.BuiltInEnvMonitorSentryEnabled) != "true" {
				middleware.InitSentry(middleware.InitSentryOption{
					Dsn:   app.Config.Monitor.Sentry.DSN,
					Debug: app.Config.Monitor.Sentry.Debug,
				})
			}
			app.Use(middleware.Sentry(func(opt *middleware.SentryOption) {
				opt.Repanic = true
				opt.WaitForDelivery = app.Config.Monitor.Sentry.WaitForDelivery
				if app.Config.Monitor.Sentry.Timeout != 0 {
					opt.Timeout = app.Config.Monitor.Sentry.Timeout
				}
			}))
		}
	})

	app.SetBeforeDestroy(func() {
		middleware.FinishSentry()
	})

	return app
}

// peerIPCaptureMiddleware stores the real TCP peer before realip rewrites
// RemoteAddr. It must run immediately before middleware.RealIP().
func peerIPCaptureMiddleware() zoox.Middleware {
	return func(ctx *zoox.Context) {
		if peer := peerip.RemoteAddrIP(ctx.Request.RemoteAddr); peer != "" {
			ctx.Request = ctx.Request.WithContext(peerip.With(ctx.Request.Context(), peer))
		}
		ctx.Next()
	}
}
