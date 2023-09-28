package core

import (
	"fmt"
	"net/http"

	"github.com/go-zoox/logger"
	"github.com/go-zoox/proxy"
	"github.com/go-zoox/zoox"
	"github.com/go-zoox/zoox/middleware"
)

func (c *core) build() error {
	// middlewares

	c.app.Use(func(ctx *zoox.Context) {
		// f, _ := os.Create("./trace.out")
		// trace.Start(f)
		// go func() {
		// 	time.Sleep(3 * time.Second)
		// 	trace.Stop()
		// 	f.Close()
		// 	time.Sleep(1 * time.Second)
		// 	os.Exit(1)
		// }()

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

		// if err := serviceIns.Validate(); err != nil {
		// 	return false, proxy.NewHTTPError(500, err.Error())
		// }

		// if ok, ips, err := c.CheckDNS(serviceIns.Hostname()); err != nil {
		// 	logger.Errorf("failed to check dns: %s", err)
		// 	return false, proxy.NewHTTPError(500, err.Error())
		// } else if !ok {
		// 	logger.Warnf("[dns] service(%s) is not ok", serviceIns.Hostname())
		// 	// return true, nil
		// 	// return false, proxy.NewHTTPError(404, err.Error())

		// 	// ctx.Status(404)
		// 	return false, nil
		// } else {
		// 	ctx.Logger.Infof("[dns] service(%s) is ok (ips: %s)", serviceIns.Hostname(), strings.Join(ips, ", "))
		// }

		// service
		// cfg.Target = serviceIns.Target()
		// cfg.Rewrites := serviceIns.Rewrite()

		cfg.OnRequest = func(req, inReq *http.Request) error {
			req.URL.Scheme = serviceIns.Protocol
			req.URL.Host = "httpbin.org" // serviceIns.Host()

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

			return nil
		}

		cfg.OnResponse = func(res *http.Response, inReq *http.Request) error {
			for k, v := range serviceIns.Response.Headers {
				ctx.Writer.Header().Set(k, v)
			}

			ctx.Writer.Header().Set("X-Proxy-By", fmt.Sprintf("gozoox-ingress/%s", c.version))
			return nil
		}

		ctx.Logger.Infof("[proxy][host: %s] %s %s => %s", hostname, method, path, serviceIns.Target())

		return
	}))

	return nil
}
