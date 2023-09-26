package core

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-zoox/ingress/core/service"
	"github.com/go-zoox/logger"
	"github.com/go-zoox/proxy"
	"github.com/go-zoox/zoox"
	"github.com/go-zoox/zoox/middleware"
)

func (c *core) build() error {
	// plugins
	c.app.Use(middleware.Proxy(func(cfg *middleware.ProxyConfig, ctx *zoox.Context) (next bool, err error) {
		cfg.OnRequest = func(req *http.Request) error {
			for _, plugin := range c.plugins {
				if err := plugin.OnRequest(ctx, ctx.Request); err != nil {
					return err
				}
			}

			return nil
		}

		cfg.OnResponse = func(res *http.Response) error {
			for _, plugin := range c.plugins {
				if err := plugin.OnResponse(ctx, ctx.Writer); err != nil {
					return err
				}
			}

			return nil
		}

		return true, nil
	}))

	// services (core plugin)
	c.app.Use(middleware.Proxy(func(cfg *middleware.ProxyConfig, ctx *zoox.Context) (next bool, err error) {
		hostname := ctx.Hostname()
		method := ctx.Method
		path := ctx.Path

		key := fmt.Sprintf("%s:%s", hostname, path)
		serviceIns := &service.Service{}
		if err := ctx.Cache().Get(key, serviceIns); err != nil {
			serviceIns, err = c.match(hostname, path)
			if err != nil {
				logger.Errorf("failed to get config: %s", err)
				//
				return false, proxy.NewHTTPError(404, "Not Found")
			}

			ctx.Cache().Set(key, serviceIns, 15*time.Second)
		}

		if serviceIns == nil {
			// return false, proxy.NewHTTPError(404, "Not Found")
			return true, nil
		}

		// service
		cfg.Target = serviceIns.Target()
		cfg.Rewrites = serviceIns.Rewrite()

		ctx.Logger.Infof("[proxy: %s] %s %s => %s", hostname, method, path, cfg.Target)

		return
	}))

	return nil
}
