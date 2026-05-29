package core

import (
	"fmt"

	"github.com/go-zoox/ingress/core/waf"
	"github.com/go-zoox/ingress/core/ratelimit"
	"github.com/go-zoox/ingress/core/security"
	"github.com/go-zoox/kv"
	"github.com/go-zoox/kv/redis"
)

func (c *core) prepare() error {
	if err := inferBackendTypes(c.cfg); err != nil {
		return err
	}
	// default config when unset
	if c.cfg.Cache.TTL == 0 {
		c.cfg.Cache.TTL = 60
	}

	if err := c.cfg.Logging.Prepare(c.cfg.Admin, c.configFilePath); err != nil {
		return fmt.Errorf("logging: %w", err)
	}
	if c.cfg.Logging.Configured() {
		c.app.Config.Logger = c.cfg.Logging.Zoox()
	}
	c.app.Config.Logger.Middleware.Disabled = true

	// prepare cache
	c.prepareCache()

	for _, plugin := range c.plugins {
		if err := plugin.Prepare(c.app, c.cfg); err != nil {
			return err
		}
	}

	var err error
	c.router, err = compileRouterIndex(c.cfg.Rules, c.cfg.Fallback)
	if err != nil {
		return fmt.Errorf("compile router: %w", err)
	}

	if err := compileAllBackendCachePathRules(c.cfg); err != nil {
		return fmt.Errorf("compile backend.cache paths: %w", err)
	}

	c.wafByRuleIdx, c.wafFallback, err = waf.CompileIngress(c.cfg.WAF, c.cfg.Rules)
	if err != nil {
		return fmt.Errorf("compile waf: %w", err)
	}

	c.rateLimits, err = ratelimit.Compile(
		c.cfg.RateLimit,
		c.cfg.Rules,
		c.cfg.Cache.Host,
		c.cfg.Cache.Port,
		c.cfg.Cache.Username,
		c.cfg.Cache.Password,
		c.cfg.Cache.DB,
		c.cfg.Cache.Prefix,
	)
	if err != nil {
		return fmt.Errorf("compile rate_limit: %w", err)
	}

	c.security, err = security.Compile(c.cfg.Security, c.cfg.Rules)
	if err != nil {
		return fmt.Errorf("compile security: %w", err)
	}

	c.errorPages, err = compileErrorPages(c.cfg)
	if err != nil {
		return fmt.Errorf("compile error_pages: %w", err)
	}

	return nil
}

func (c *core) prepareCache() {
	if c.cfg.Cache.Host != "" {
		prefix := c.cfg.Cache.Prefix
		if prefix == "" {
			prefix = "gozoox-ingress:"
		}

		c.app.Config.Cache = kv.Config{
			Engine: "redis",
			Config: &redis.Config{
				Host:     c.cfg.Cache.Host,
				Port:     int(c.cfg.Cache.Port),
				Username: c.cfg.Cache.Username,
				Password: c.cfg.Cache.Password,
				DB:       int(c.cfg.Cache.DB),
				Prefix:   prefix,
			},
		}
	}

	if err := c.app.Cache().Clear(); err != nil {
		c.app.Logger().Errorf("[prepareCache] failed to clear cache: %s", err)
	}
}
