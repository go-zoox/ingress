package core

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-zoox/ingress"
	"github.com/go-zoox/ingress/core"
	"github.com/go-zoox/proxy"
	"github.com/go-zoox/proxy/utils/rewriter"
	"github.com/go-zoox/zoox"
)

type Core struct {
	Application *core.Core
}

type Config struct {
	Host            string
	Path            string
	ServiceName     string
	ServicePort     int64
	ServiceProtocol string
	Request         core.ConfigRequest
	Response        core.ConfigResponse
}

type SSLConfig struct {
	Domain string
	//
	Certificate    string
	CertificateKey string
}

func (c *Core) getConfig(host string, path string) *Config {
	var cfg Config

	// rule
	for _, rule := range c.Application.Config.Rules {
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

	return &cfg
}

func (c *Core) GetSSLConfig(domain string) *SSLConfig {
	var cfg SSLConfig

	// ssl
	if c.Application.Config.SSL != nil {
		parts := strings.Split(domain, ".")

		for _, ssl := range c.Application.Config.SSL {
			// extract match domain
			// x.y.com.cn => x.y.com
			// x.y.z.com => x.y.z.com
			if ssl.Domain == domain {
				cfg.Certificate = ssl.Cert.Certificate
				cfg.CertificateKey = ssl.Cert.CertificateKey
			} else {
				// x.y.com => y.com
				// x.y.z.com => y.z.com => z.com
				for i := 1; i <= len(parts)-2; i++ {
					if ssl.Domain == strings.Join(parts[i:], ".") {
						cfg.Certificate = ssl.Cert.Certificate
						cfg.CertificateKey = ssl.Cert.CertificateKey
						break
					}
				}
			}
		}
	}

	return &cfg
}

func (c *Core) OnRequest(ctx *zoox.Context, req *http.Request) error {
	ctx.Logger.Infof("[service][%s] %s %s", ctx.Hostname(), ctx.Method, ctx.Path)

	cfg := c.getConfig(ctx.Hostname(), ctx.Path)
	if cfg.ServiceName == "" || cfg.ServicePort == int64(0) {
		return proxy.NewHTTPError(404, "Not Found")
	}

	ctx.Request.URL.Scheme = cfg.ServiceProtocol
	ctx.Request.URL.Host = fmt.Sprintf("%s:%d", cfg.ServiceName, cfg.ServicePort)

	if cfg.Request.Rewrites != nil {
		rewriters := &rewriter.Rewriters{}
		for _, r := range cfg.Request.Rewrites {
			ft := strings.Split(r, ":")
			rewriters.Add(ft[0], ft[1])
		}

		ctx.Request.URL.Path = rewriters.Rewrite(ctx.Path)
	}

	if cfg.Request.Query != nil {
		originQuery := ctx.Request.URL.Query()
		for k, v := range cfg.Request.Query {
			originQuery[k] = []string{v}
		}

		ctx.Request.URL.RawQuery = originQuery.Encode()
	}

	if cfg.Request.Headers != nil {
		for k, v := range cfg.Request.Headers {
			ctx.Request.Header.Set(k, v)
		}
	}

	ctx.Logger.Debugf("request: %s://%s%s", ctx.Request.URL.Scheme, ctx.Request.URL.Host, ctx.Request.URL.Path)

	return nil
}

func (c *Core) OnResponse(ctx *zoox.Context, res http.ResponseWriter) error {
	ctx.Logger.Info("[service][%s][%s] %s %s %d +%d ms", ctx.Host, ctx.Request.URL.Host, ctx.Method, ctx.Path, ctx.Status, 123456)
	ctx.Set("X-Proxy", fmt.Sprintf("go-zoox_ingress/%s", ingress.Version))
	ctx.Set("X-Request-ID", ctx.RequestID())
	return nil
}
