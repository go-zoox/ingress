package core

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	ingress "github.com/go-zoox/ingress/core"
	"github.com/go-zoox/proxy"
	"github.com/go-zoox/proxy/utils/rewriter"
)

type Core struct {
	Application *ingress.Application
}

type Config struct {
	Host          string
	Path          string
	ServiceName   string
	ServicePort   int64
	ServiceScheme string
	Request       ingress.ConfigRequest
	Response      ingress.ConfigResponse
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
						cfg.ServiceScheme = rpath.Backend.ServiceScheme
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
				cfg.ServiceScheme = rule.Backend.ServiceScheme
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

func (c *Core) OnRequest(ctx *ingress.Context) error {
	cfg := c.getConfig(ctx.Host, ctx.Path)
	if cfg.ServiceName == "" || cfg.ServicePort == int64(0) {
		panic(proxy.NewHTTPError(404, "Not Found"))
	}

	ctx.Request.URL.Host = fmt.Sprintf("%s:%d", cfg.ServiceName, cfg.ServicePort)

	if cfg.Request.Rewrites != nil {
		rewriters := rewriter.Rewriters{}
		for _, r := range cfg.Request.Rewrites {
			ft := strings.Split(r, ":")
			rewriters = append(rewriters, &rewriter.Rewriter{
				From: ft[0],
				To:   ft[1],
			})
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

	return nil
}

func (c *Core) OnResponse(ctx *ingress.Context) error {
	reqestTime := ctx.RequestTime() / time.Millisecond

	ctx.Logger.Info("[service][%s][%s] %s %s %d +%d ms", ctx.Host, ctx.Request.URL.Host, ctx.Method, ctx.Path, ctx.Status, reqestTime)

	ctx.Response.Header.Set("x-powered-by", fmt.Sprintf("go-zoox/ingress v%s", c.Application.Version))
	ctx.Response.Header.Set("x-request-id", ctx.RequestId)
	ctx.Response.Header.Set("x-request-time", fmt.Sprintf("%d ms", reqestTime))

	return nil
}
