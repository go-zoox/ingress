package core

import (
	"fmt"
	"os"

	"github.com/go-zoox/ingress/core/ratelimit"
	"github.com/go-zoox/ingress/core/security"
	"github.com/go-zoox/ingress/core/waf"
	"github.com/go-zoox/kv"
	"github.com/go-zoox/kv/redis"
	"github.com/go-zoox/zoox"
)

func (c *core) prepare() error {
	if err := inferBackendTypes(c.cfg); err != nil {
		return err
	}
	// default config when unset
	if c.cfg.Cache.TTL == 0 {
		c.cfg.Cache.TTL = 60
	}

	// Use a stable session encryption key for OAuth2/OIDC across replicas.
	// Precedence: SECRET_KEY env > config secret_key > zoox per-process default.
	c.applySecretKey()

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

	c.maintenanceByRule, err = compileMaintenanceByRule(c.cfg)
	if err != nil {
		return fmt.Errorf("compile maintenance: %w", err)
	}

	c.globalMaintenance, err = compileGlobalMaintenance(c.cfg.Maintenance)
	if err != nil {
		return fmt.Errorf("compile global maintenance: %w", err)
	}

	c.ingressStatusPath, err = compileIngressStatusPath(c.cfg.Maintenance.StatusPath)
	if err != nil {
		return fmt.Errorf("maintenance.status_path: %w", err)
	}

	return nil
}

// applySecretKey sets the session encryption key used for OAuth2/OIDC session cookies.
// The SECRET_KEY env var takes precedence over the config value, so operators can rotate
// the secret without editing the config file, and so the same value can be shared across
// every replica (required for OAuth2 to survive load-balanced callbacks). When both are
// absent we leave app.Config.SecretKey empty and let zoox use its per-process random key.
func (c *core) applySecretKey() {
	secretKey := c.cfg.SecretKey
	if v := os.Getenv(zoox.BuiltInEnvSecretKey); v != "" {
		secretKey = v
	}
	if secretKey != "" {
		c.app.Config.SecretKey = secretKey
	}
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
