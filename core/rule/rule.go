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
	// HostType is the host match type: exact, regex, wildcard, or auto (empty).
	// Empty or "auto" selects exact vs regex vs wildcard from Host at compile time.
	// Set "exact" explicitly to match Host as a literal string even if it looks like a pattern.
	HostType string `config:"host_type"`
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
}

type Redirect struct {
	URL string `config:"url"`
	// Permanent selects 301/308 vs 302/307 depending on WithOriginMethodAndBody.
	Permanent bool `config:"permanent"`
	// WithOriginMethodAndBody uses HTTP 307/308 so clients preserve method and body on redirect.
	// Default false uses 302/301 via RedirectTemporary / RedirectPermanent.
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
