package core

// References:
//   https://www.cnblogs.com/zyndev/p/14454891.html
//   https://h1z3y3.me/posts/simple-and-powerful-reverse-proxy-in-golang/
//   https://segmentfault.com/a/1190000039778241

import (
	"fmt"

	"github.com/go-zoox/ingress/core/waf"
	"github.com/go-zoox/zoox"
	"github.com/go-zoox/zoox/defaults"
)

type Core interface {
	Version() string
	Run() error
	//
	Reload(cfg *Config) error
	ReloadFromFile() error
}

type core struct {
	app *zoox.Application

	version string
	cfg     *Config

	configFilePath string
	pidFilePath    string

	router *routerIndex

	wafByRuleIdx []*waf.Profile
	wafFallback  *waf.Profile

	plugins []Plugin
}

func New(version string, cfg *Config) (Core, error) {
	return NewWithPaths(version, cfg, "", "")
}

// NewWithPaths creates core with ingress config and pid file paths (required for admin console).
func NewWithPaths(version string, cfg *Config, configFilePath, pidFilePath string) (Core, error) {
	c := &core{
		app: defaults.Default(),
		//
		version: version,
		//
		cfg: cfg,
		//
		configFilePath: configFilePath,
		pidFilePath:    pidFilePath,
	}

	if err := c.prepare(); err != nil {
		return nil, fmt.Errorf("failed to prepare: %s", err)
	}

	return c, nil
}

func (c *core) Version() string {
	return c.version
}
