package app

import (
	"fmt"
	"net/http"

	"github.com/go-zoox/ingress/admin/console/bootstrap"
	"github.com/go-zoox/ingress/admin/console/config"
	"github.com/go-zoox/ingress/admin/console/handler"
	"github.com/go-zoox/ingress/admin/console/static"
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
	app.Config.Port = int(cfg.Port)

	api := handler.NewAPI(cfg)
	g := app.Group("/api/v1")
	api.Mount(g)

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
