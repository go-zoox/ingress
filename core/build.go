package core

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-zoox/logger"
	"github.com/go-zoox/proxy"
	"github.com/go-zoox/zoox"
	"github.com/go-zoox/zoox/middleware"
)

func (c *core) build() error {
	// config
	c.app.Config.Port = int(c.cfg.Port)
	c.app.Config.HTTPSPort = int(c.cfg.HTTPS.Port)

	// middlewares
	c.app.Use(func(ctx *zoox.Context) {
		if c.cfg.HealthCheck.Outer.Enable {
			if ctx.Path == c.cfg.HealthCheck.Outer.Path {
				if c.cfg.HealthCheck.Outer.Ok {
					ctx.String(http.StatusOK, "ok")
					return
				}
			}
		}

		ctx.Next()
	})

	// services (core plugin)
	c.app.Use(middleware.Proxy(func(ctx *zoox.Context, cfg *middleware.ProxyConfig) (next, stop bool, err error) {
		hostname := ctx.Hostname()
		method := ctx.Method
		path := ctx.Path

		serviceIns, rule, err := c.match(ctx, hostname, path)
		if err != nil {
			logger.Errorf("failed to match rule (host: %s, path: %s): %s", hostname, path, err)

			// service not found
			return false, false, proxy.NewHTTPError(404, "Not Found")
		}

		// redirect
		if rule.Backend.Redirect.URL != "" {
			// if !rule.Backend.Redirect.Permanent {
			// 	ctx.RedirectTemporary(rule.Backend.Redirect.URL)
			// } else {
			// 	ctx.RedirectPermanent(rule.Backend.Redirect.URL)
			// }

			ctx.Redirect(rule.Backend.Redirect.URL)

			return false, true, nil
		}

		if err := serviceIns.Validate(); err != nil {
			return false, false, proxy.NewHTTPError(500, err.Error())
		}

		ips, err := serviceIns.CheckDNS()
		if err != nil {
			logger.Errorf("check dns error: %s", err)

			// exact service specify
			if rule.HostType == "exact" {
				return false, false, proxy.NewHTTPError(503, "Service Unavailable")
			}

			// regular expression service specify, maybe the service is not found
			return false, false, proxy.NewHTTPError(404, "Service Not Found")
		}

		ctx.Logger.Debugf("[dns] service(%s) is ok (ips: %s)", serviceIns.Name, strings.Join(ips, ", "))

		cfg.OnRequest = func(req, inReq *http.Request) error {
			req.URL.Scheme = serviceIns.Protocol
			req.URL.Host = serviceIns.Host()

			// apply host
			if serviceIns.Request.Host.Rewrite {
				req.Host = serviceIns.Host()
			}

			// apply path
			req.URL.Path = serviceIns.Rewrite(req.URL.Path)

			// apply headers
			for k, v := range serviceIns.Request.Headers {
				req.Header.Set(k, v)
			}

			// apply query
			if serviceIns.Request.Query != nil {
				originQuery := req.URL.Query()
				for k, v := range serviceIns.Request.Query {
					originQuery.Set(k, v)
				}
				req.URL.RawQuery = originQuery.Encode()
			}

			// plugins
			for _, plugin := range c.plugins {
				if err := plugin.OnRequest(ctx, req); err != nil {
					return err
				}
			}

			return nil
		}

		cfg.OnResponse = func(res *http.Response, inReq *http.Request) error {
			for k, v := range serviceIns.Response.Headers {
				ctx.Writer.Header().Set(k, v)
			}

			// plugins
			for _, plugin := range c.plugins {
				if err := plugin.OnResponse(ctx, res); err != nil {
					return err
				}
			}

			ctx.Writer.Header().Del("X-Powered-By")
			ctx.Writer.Header().Set("X-Powered-By", fmt.Sprintf("gozoox-ingress/%s", c.version))
			return nil
		}

		ctx.Logger.Infof("[host: %s, target: %s] \"%s %s %s\" %d", hostname, serviceIns.Target(), method, path, ctx.Request.Proto, ctx.StatusCode())

		return
	}))

	return nil
}
