package rule

import (
	"github.com/go-zoox/ingress/core/service"
)

type Rule struct {
	Host string `config:"host"`
	// WAFPatch is filled from rules[].waf by waf.ApplyRulePatchesFromYAML (the tag loader skips this field).
	WAFPatch map[string]any `config:"-"`
	Backend  Backend        `config:"backend"`
	//
	Paths []Path `config:"paths"`
	// RateLimit optional per-route throttling; global rate_limit in config also applies.
	RateLimit RateLimit `config:"rate_limit"`
	// Security optional per-route security response headers; global security also applies as baseline.
	Security Security `config:"security"`
	// HostType is the host match type: exact, regex, wildcard, or auto (empty).
	// Empty or "auto" selects exact vs regex vs wildcard from Host at compile time.
	// Set "exact" explicitly to match Host as a literal string even if it looks like a pattern.
	HostType string `config:"host_type"`
	// Enabled controls whether the rule participates in routing. Omitted or true = enabled.
	Enabled *bool `config:"enabled"`
	// ConnectProxy enables forward-proxy handling for this proxy hostname:
	// both CONNECT (HTTPS targets) and absolute-form HTTP requests
	// (GET http://target/...). For both forms the HTTP Host is the destination,
	// so routing uses the TLS SNI hostname after ingress terminates TLS.
	// The normal rule backend points to the next plain-HTTP proxy hop
	// (for example anchor-backend:3128).
	ConnectProxy ConnectProxy `config:"connect_proxy"`
}

// ConnectProxy configures CONNECT tunnelling for a TLS-terminated proxy hostname.
type ConnectProxy struct {
	// Enabled turns this rule into a forward-proxy route matched by TLS SNI.
	Enabled bool `config:"enabled"`
	// AllowedTargets contains optional regular expressions matched against the
	// canonical destination authority. CONNECT supplies host:port directly;
	// HTTP absolute-form targets get an implicit :80. Empty means all targets
	// are allowed. Patterns are full-string matches.
	AllowedTargets []string `config:"allowed_targets"`
	// Upstream is the next HTTP proxy hop for CONNECT/absolute-form requests.
	// When empty, the rule backend.service is used for backward compatibility.
	Upstream service.Service `config:"upstream"`
	// ConnectTimeout is the dial/HTTP CONNECT write timeout in seconds. Default 10.
	ConnectTimeout int64 `config:"connect_timeout"`
	// ResponseTimeout is the upstream CONNECT response header timeout in seconds. Default 10.
	ResponseTimeout int64 `config:"response_timeout"`
	// IdleTimeout is the tunnel idle timeout in seconds. Default 600; 0 uses the default.
	IdleTimeout int64 `config:"idle_timeout"`
	// MaxConnections caps concurrent CONNECT/HTTP proxy requests for this route.
	// 0 means unlimited.
	MaxConnections int64 `config:"max_connections"`
}

// Backend describes what to do for a matched host or path (rules[].backend or paths[].backend;
// fallback uses the same shape).
//
// YAML backend.type must be one of: "service", "handler", or "redirect". When omitted, ingress
// infers the type if exactly one mode is clearly configured:
//
//   - backend.redirect.url present → "redirect"
//
//   - non-empty backend.handler fields → "handler"
//
//   - otherwise → "service" (upstream via backend.service)
//
// If two or more modes look configured at once with type omitted, validation fails and requires an
// explicit backend.type.
//
// Each explicit type allows only its own block: "service" tolerates backend.service only;
// "handler" tolerates backend.handler only; "redirect" tolerates backend.redirect only.
type Backend struct {
	Type string `config:"type"`
	// Cache enables optional HTTP response caching for service, handler, and redirect backends; see rule.BackendCache.Enabled.
	Cache BackendCache `config:"cache"`
	// Mode is internal (default) or external for service upstreams. Prefer backend.service.mode;
	// this field is a legacy alias when service.mode is empty. Empty means "inherit from service.mode only".
	Mode string `config:"mode"`
	//
	Service service.Service `config:"service"`
	//
	Handler Handler `config:"handler"`
	//
	Redirect Redirect `config:"redirect"`
}

type Path struct {
	Path    string  `config:"path"`
	Backend Backend `config:"backend"`
	// Security optional per-path security response headers; global and rules[].security apply as baseline.
	Security Security `config:"security"`
}

type Redirect struct {
	URL string `config:"url"`
	// Duration is permanent or temporary (default temporary when empty and permanent is false).
	Duration string `config:"duration"`
	// PreserveRequest uses HTTP 307/308 so clients keep the original method and body.
	PreserveRequest bool `config:"preserve_request"`
	// PreservePath keeps the incoming request path and query when the redirect url has no path of its own.
	//
	// Only meaningful on paths[].backend.redirect: a path-level redirect without preserve_path jumps to the
	// configured url as written (no request path/query appended); set preserve_path: true to restore the
	// whole-site behaviour. Host-level (rules[].backend.redirect) and fallback redirects always preserve
	// the request path and query, so preserve_path: true there has no effect and validate rejects it.
	PreservePath bool `config:"preserve_path"`
	// Permanent selects 301/308 vs 302/307 when Duration is empty (legacy).
	Permanent bool `config:"permanent"`
	// WithOriginMethodAndBody is a legacy alias for preserve_request (deprecated).
	WithOriginMethodAndBody bool `config:"with_origin_method_and_body"`
}

// Handler is used only when Backend.Type == "handler".
//
// Handler.Type (YAML handler.type) selects the implementation: static_response, file_server,
// templates, or script — see core/constants.go handlerType* and core/build.go switch.
type Handler struct {
	Type       string            `config:"type,default=static_response"`
	Engine     string            `config:"engine,default=javascript"`
	Script     string            `config:"script"`
	StatusCode int64             `config:"status_code,default=200"`
	Headers    map[string]string `config:"headers"`
	Body       string            `config:"body"`
	RootDir    string            `config:"root_dir"`
	IndexFile  string            `config:"index_file,default=index.html"`
}
