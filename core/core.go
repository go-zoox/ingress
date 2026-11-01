package core

// References:
//   https://www.cnblogs.com/zyndev/p/14454891.html
//   https://h1z3y3.me/posts/simple-and-powerful-reverse-proxy-in-golang/
//   https://segmentfault.com/a/1190000039778241

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/go-zoox/ingress/core/ratelimit"
	"github.com/go-zoox/ingress/core/security"
	"github.com/go-zoox/ingress/core/waf"
	"github.com/go-zoox/zoox"
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
	SetAccessMetricsCallback(cb AccessMetricsCallback)
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

	connectProxyRules  []compiledConnectProxyRule
	httpProxyTransport *http.Transport

	wafByRuleIdx []*waf.Profile
	wafFallback  *waf.Profile

	rateLimits *ratelimit.Ingress

	security *security.Ingress

	errorPages *compiledErrorPages

	maintenanceByRule []compiledRuleMaintenance

	globalMaintenance compiledGlobalMaintenance

	ingressStatusPath string

	plugins []Plugin

	wafRuntimeOverride *bool
	wafOverrideMu      sync.RWMutex
	wafCallback        WAFCallback
	accessMetricsCb    AccessMetricsCallback

	runtimeConfigHash string
}

func New(version string, cfg *Config) (Core, error) {
	return NewWithPaths(version, cfg, "", "")
}

// NewWithPaths creates core with ingress config and pid file paths (required for admin console).
func NewWithPaths(version string, cfg *Config, configFilePath, pidFilePath string) (Core, error) {
	c := &core{
		app: newIngressApplication(),
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

// IsWAFEnabled returns true if the global WAF baseline should be active
// (considering config + runtime override). Per-route WAF can still be enabled
// through rules[].waf.enabled when the global baseline is off.
func (c *core) IsWAFEnabled() bool {
	c.wafOverrideMu.RLock()
	override := c.wafRuntimeOverride
	c.wafOverrideMu.RUnlock()
	if override != nil {
		return *override
	}
	return c.cfg.WAF.Enabled
}

// isWAFProfileActive reports whether a compiled per-route/fallback WAF profile
// should run. A nil runtime override means "follow the profile" (which already
// reflects the merged global + per-route enabled flags); an explicit override
// still wins for emergency global off/on.
func (c *core) isWAFProfileActive(p *waf.Profile) bool {
	if p == nil || !p.Enabled {
		return false
	}
	c.wafOverrideMu.RLock()
	override := c.wafRuntimeOverride
	c.wafOverrideMu.RUnlock()
	if override != nil {
		return *override
	}
	return true
}

// SetWAFCallback registers a callback invoked when WAF blocks or audits.
func (c *core) SetWAFCallback(cb WAFCallback) {
	c.wafCallback = cb
}

// SetAccessMetricsCallback registers a callback for structured access log metrics.
func (c *core) SetAccessMetricsCallback(cb AccessMetricsCallback) {
	c.accessMetricsCb = cb
}
