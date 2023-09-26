package core

import (
	"net/http"

	"github.com/go-zoox/logger"
	"github.com/go-zoox/proxy"
	"github.com/go-zoox/zoox"
	"github.com/go-zoox/zoox/middleware"
)

func (c *core) build() error {
	// plugins
	c.app.Use(middleware.Proxy(func(ctx *zoox.Context, cfg *middleware.ProxyConfig) (next bool, err error) {
		cfg.OnRequest = func(req, inReq *http.Request) error {
			for _, plugin := range c.plugins {
				if err := plugin.OnRequest(ctx, ctx.Request); err != nil {
					return err
				}
			}

			return nil
		}

		cfg.OnResponse = func(res *http.Response, inReq *http.Request) error {
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
	c.app.Use(middleware.Proxy(func(ctx *zoox.Context, cfg *middleware.ProxyConfig) (next bool, err error) {
		hostname := ctx.Hostname()
		method := ctx.Method
		path := ctx.Path

		serviceIns, err := c.match(ctx, hostname, path)
		if err != nil {
			logger.Errorf("failed to get config: %s", err)
			//
			return false, proxy.NewHTTPError(404, "Not Found")
		}

		if serviceIns == nil {
			// return false, proxy.NewHTTPError(404, "Not Found")
			return true, nil
		}

		// service
		// cfg.Target = serviceIns.Target()
		// cfg.Rewrites := serviceIns.Rewrite()

		cfg.OnRequest = func(req, inReq *http.Request) error {
			req.URL.Scheme = serviceIns.Protocol
			req.URL.Host = serviceIns.URLHost()
			req.URL.Path = serviceIns.Rewrite(req.URL.Path)

			if serviceIns.Request.Host.Rewrite {
				req.Host = serviceIns.URLHost()
			}

			return nil
		}

		ctx.Logger.Infof("[proxy: %s] %s %s => %s", hostname, method, path, serviceIns.Target())

		return
	}))

	return nil
}
