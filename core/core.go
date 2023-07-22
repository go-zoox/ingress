package core

// References:
//   https://www.cnblogs.com/zyndev/p/14454891.html
//   https://h1z3y3.me/posts/simple-and-powerful-reverse-proxy-in-golang/
//   https://segmentfault.com/a/1190000039778241

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-zoox/proxy"
	"github.com/go-zoox/zoox"
	"github.com/go-zoox/zoox/defaults"
	"github.com/go-zoox/zoox/middleware"
)

type Core struct {
	*zoox.Application

	Version string
	Config  *Config

	plugins []Plugin
}

func New(version string, cfg *Config) *Core {
	return &Core{
		Application: defaults.Default(),
		//
		Version: version,
		Config:  cfg,
	}
}

// func (app *Application) ServeHTTP(w http.ResponseWriter, req *http.Request) {
// 	// health check
// 	if req.URL.Path == "/health" {
// 		w.WriteHeader(http.StatusOK)
// 		return
// 	}

// 	ctx := app.createContext()

// 	p := proxy.New(&proxy.Config{
// 		OnRequest: func(req *http.Request) error {
// 			ctx.Request = NewRequest(req)

// 			host, _ := proxy.ParseHostPort(ctx.Request.Host)
// 			ctx.Host = host
// 			ctx.Path = req.URL.Path
// 			ctx.Method = req.Method
// 			ctx.RequestId = uuid.V4()
// 			ctx.RequestStartAt = time.Now()

// 			// logger.Info("[request] %s %s", req.Method, req.URL.String())

// 			ctx.onRequest(ctx)

// 			return nil
// 		},
// 		OnResponse: func(res *http.Response) error {
// 			ctx.Response = NewResponse(res)

// 			ctx.Status = res.StatusCode
// 			ctx.RequestEndAt = time.Now()
// 			// ctx.Logger.Info("[response] %d %s", res.StatusCode, res.Status)

// 			ctx.onResponse(ctx)
// 			return nil
// 		},
// 	})

// 	p.ServeHTTP(w, req)
// }

func (c *Core) Plugin(plugin ...Plugin) *Core {
	c.plugins = append(c.plugins, plugin...)
	return c
}

func (c *Core) Build() error {
	c.Application.Use(middleware.Proxy(func(cfg *middleware.ProxyConfig, ctx *zoox.Context) (next bool, err error) {
		serviceCfg := c.getConfig(ctx.Hostname(), ctx.Path)
		if serviceCfg.ServiceName == "" || serviceCfg.ServicePort == int64(0) {
			return false, proxy.NewHTTPError(404, "Not Found")
		}

		cfg.Target = fmt.Sprintf("%s://%s:%d", serviceCfg.ServiceProtocol, serviceCfg.ServiceName, serviceCfg.ServicePort)

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

		ctx.Logger.Debugf("[proxy] %s %s => %s", ctx.Method, ctx.Path, cfg.Target)

		return
	}))

	return nil
}

// func (c *Core) Start() error {
// 	return c.Run()
// }

func (c *Core) Start() error {
	if err := c.Build(); err != nil {
		return err
	}

	https := c.Config.HTTPSPort != 0
	httpsPort := c.Config.HTTPSPort

	port := c.Config.Port
	if port == 0 {
		port = 8080
	}

	c.Logger.Info("ingress start at http://127.0.0.1:%d", port)

	if https {
		c.Logger.Info("ingress start at https://127.0.0.1:%d", httpsPort)

		// http.ListenAndServeTLS(fmt.Sprintf(""::"%d", port), "", "", app)
		// srv := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: app}
		var minVersionTLS uint16 = 0x0301 // TLS 1.0
		server := &http.Server{
			ReadTimeout:  50 * time.Second,
			WriteTimeout: 600 * time.Second,
			IdleTimeout:  60 * time.Second,
			Addr:         fmt.Sprintf(":%d", httpsPort),
			TLSConfig: &tls.Config{
				MinVersion: minVersionTLS,
				GetCertificate: func(chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
					if chi.ServerName == "" {
						return nil, fmt.Errorf("no server name")
					}

					// ssl
					if c.Config.SSL != nil {
						var certificate string
						var certificateKey string

						parts := strings.Split(chi.ServerName, ".")

						for _, ssl := range c.Config.SSL {
							// extract match domain
							// x.y.com.cn => x.y.com
							// x.y.z.com => x.y.z.com
							if ssl.Domain == chi.ServerName {
								certificate = ssl.Cert.Certificate
								certificateKey = ssl.Cert.CertificateKey
							} else {
								// x.y.com => y.com
								// x.y.z.com => y.z.com => z.com
								for i := 1; i <= len(parts)-2; i++ {
									if ssl.Domain == strings.Join(parts[i:], ".") {
										certificate = ssl.Cert.Certificate
										certificateKey = ssl.Cert.CertificateKey
										break
									}
								}
							}
						}

						if certificate == "" || certificateKey == "" {
							return nil, fmt.Errorf("no certificate")
						}

						cert, err := tls.LoadX509KeyPair(certificate, certificateKey)
						if err != nil {
							return nil, err
						}

						return &cert, nil
					}

					return nil, nil
				},
			},
			Handler: c,
		}

		go func() {
			if err := server.ListenAndServeTLS("", ""); err != nil {
				panic(err)
			}
		}()
	}

	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), c); err != nil {
		return err
	}

	return nil
}

func (c *Core) getConfig(host string, path string) (cfg *ServiceConfig) {
	cfg = &ServiceConfig{}

	// rule
	for _, rule := range c.Config.Rules {
		// logger.Info("rule: %s: %s:%d", rule.Host, rule.Backend.ServiceName, rule.Backend.ServicePort)
		if rule.Host == host {
			if rule.Paths != nil {
				for _, rpath := range rule.Paths {
					if isMatched, err := regexp.MatchString(rpath.Path, path); err == nil && isMatched {
						cfg.Host = rule.Host
						cfg.Path = rpath.Path
						cfg.ServiceProtocol = rpath.Backend.ServiceProtocol
						cfg.ServiceName = rpath.Backend.ServiceName
						cfg.ServicePort = rpath.Backend.ServicePort
						cfg.Request = rpath.Backend.Request
						cfg.Response = rpath.Backend.Response
						break
					}
				}
			}

			if cfg.ServiceName == "" {
				cfg.Host = rule.Host
				cfg.ServiceProtocol = rule.Backend.ServiceProtocol
				cfg.ServiceName = rule.Backend.ServiceName
				cfg.ServicePort = rule.Backend.ServicePort
				cfg.Request = rule.Backend.Request
				cfg.Response = rule.Backend.Response
			}
		}
	}

	return cfg
}
