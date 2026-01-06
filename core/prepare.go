package core

import (
	"github.com/go-zoox/kv"
	"github.com/go-zoox/kv/redis"
	"github.com/go-zoox/logger"
)

func (c *core) prepare() error {
	// 补全配置
	if c.cfg.Cache.TTL == 0 {
		c.cfg.Cache.TTL = 60
	}

	// 补全配置
	c.app.Config.Logger.Middleware.Disabled = true

	// prepare cache
	c.prepareCache()

	for _, plugin := range c.plugins {
		if err := plugin.Prepare(c.app, c.cfg); err != nil {
			return err
		}
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
		logger.Errorf("[prepareCache] failed to clear cache: %s", err)
	}
}
