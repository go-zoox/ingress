package core

// References:
//   https://www.cnblogs.com/zyndev/p/14454891.html
//   https://h1z3y3.me/posts/simple-and-powerful-reverse-proxy-in-golang/
//   https://segmentfault.com/a/1190000039778241

import (
	"fmt"
	"sync"

	"github.com/go-zoox/ingress/core/waf"
	"github.com/go-zoox/ingress/core/ratelimit"
	"github.com/go-zoox/zoox"
	"github.com/go-zoox/zoox/defaults"
)

type Core interface {
	Version() string
	Run() error
	//
	Reload(cfg *Config) error
	ReloadFromFile() error
	//
	SetWAFOverride(enabled *bool)
	GetWAFOverride() *bool
	IsWAFEnabled() bool
	//
	SetWAFCallback(cb WAFCallback)
	// ConfigFingerprint is the hash of config active in the running ingress process (updated on Reload).
	ConfigFingerprint() string
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

	rateLimits *ratelimit.Ingress

	plugins []Plugin

	wafRuntimeOverride *bool
	wafOverrideMu      sync.RWMutex
	wafCallback        WAFCallback

	runtimeConfigHash string
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
	c.refreshRuntimeConfigHash()

	return c, nil
}

func (c *core) Version() string {
	return c.version
}

// SetWAFOverride sets a runtime override for WAF enabled state.
// nil = use config file value, true = force enabled, false = force disabled.
func (c *core) SetWAFOverride(enabled *bool) {
	c.wafOverrideMu.Lock()
	defer c.wafOverrideMu.Unlock()
	c.wafRuntimeOverride = enabled
}

// GetWAFOverride returns the current runtime override.
func (c *core) GetWAFOverride() *bool {
	c.wafOverrideMu.RLock()
	defer c.wafOverrideMu.RUnlock()
	return c.wafRuntimeOverride
}

// IsWAFEnabled returns true if WAF should be active (considering config + runtime override).
func (c *core) IsWAFEnabled() bool {
	c.wafOverrideMu.RLock()
	override := c.wafRuntimeOverride
	c.wafOverrideMu.RUnlock()
	if override != nil {
		return *override
	}
	return c.cfg.WAF.Enabled
}

// SetWAFCallback registers a callback invoked when WAF blocks or audits.
func (c *core) SetWAFCallback(cb WAFCallback) {
	c.wafCallback = cb
}
