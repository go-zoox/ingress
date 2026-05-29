package core

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-zoox/ingress/core/ratelimit"
	"github.com/go-zoox/ingress/core/security"
	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/ingress/core/waf"
)

// ValidateConfig performs static configuration checks without starting servers
// or touching external systems.
func ValidateConfig(cfg *Config) error {
	if err := inferBackendTypes(cfg); err != nil {
		return err
	}

	if _, err := compileRouterIndex(cfg.Rules, cfg.Fallback); err != nil {
		return fmt.Errorf("router rules: %w", err)
	}

	if _, _, err := waf.CompileIngress(cfg.WAF, cfg.Rules); err != nil {
		return fmt.Errorf("waf: %w", err)
	}

	if err := validateRateLimit(cfg.RateLimit, "rate_limit"); err != nil {
		return err
	}
	if _, err := ratelimit.Compile(cfg.RateLimit, cfg.Rules, cfg.Cache.Host, cfg.Cache.Port, cfg.Cache.Username, cfg.Cache.Password, cfg.Cache.DB, cfg.Cache.Prefix); err != nil {
		return fmt.Errorf("rate_limit: %w", err)
	}

	if _, err := security.Compile(cfg.Security, cfg.Rules); err != nil {
		return fmt.Errorf("security: %w", err)
	}

	if err := validateErrorPages(cfg); err != nil {
		return err
	}

	if cfg.HTTPS.Port != 0 && len(cfg.HTTPS.SSL) == 0 {
		return fmt.Errorf("https.ssl is required when https.port is set")
	}

	for i, ssl := range cfg.HTTPS.SSL {
		if ssl.Domain == "" {
			return fmt.Errorf("https.ssl[%d].domain is required", i)
		}
		if ssl.Cert.Certificate == "" {
			return fmt.Errorf("https.ssl[%d].cert.certificate is required", i)
		}
		if ssl.Cert.CertificateKey == "" {
			return fmt.Errorf("https.ssl[%d].cert.certificate_key is required", i)
		}
	}

	for i := range cfg.Rules {
		r := cfg.Rules[i]
		if err := validateRateLimit(r.RateLimit, fmt.Sprintf("rules[%d]", i)); err != nil {
			return err
		}
		if err := validateBackend(r.Backend, i, r.Host, "/"); err != nil {
			return err
		}

		for j := range r.Paths {
			p := r.Paths[j]
			pathPattern := p.Path
			if pathPattern == "" {
				pathPattern = "paths[" + strconv.Itoa(j) + "]"
			}
			if err := validateBackend(p.Backend, i, r.Host, pathPattern); err != nil {
				return err
			}
		}
	}

	if strings.TrimSpace(cfg.Fallback.Service.Name) != "" {
		if err := validateBackend(cfg.Fallback, -1, "", "/"); err != nil {
			return err
		}
	}

	if cfg.Admin.Enabled {
		if cfg.Admin.Port <= 0 {
			return fmt.Errorf("admin.port must be positive when admin.enabled is true")
		}
		d := strings.ToLower(strings.TrimSpace(cfg.Admin.Database.Driver))
		switch d {
		case "sqlite", "sqlite3", "mysql", "postgres", "postgresql", "":
		default:
			return fmt.Errorf("unsupported admin.database.driver %q", cfg.Admin.Database.Driver)
		}
	}

	return nil
}

// validateBackend checks one backend under a rule.
// host is the rule's host pattern; pathPattern is paths[].path from config for path backends,
// "/" for the rule-level backend. If paths[].path is empty, messages use paths[index] as fallback.
//
// Expected backend.Type values (after inferBackendTypes): "service", "handler", or "redirect".
func validateBackend(backend rule.Backend, ruleIdx int, host, pathPattern string) error {
	svcMode := strings.TrimSpace(backend.Service.Mode)
	bkMode := strings.TrimSpace(backend.Mode)
	if svcMode != "" && bkMode != "" && svcMode != bkMode {
		return fmt.Errorf("%s: service.mode %q conflicts with backend.mode %q (use one field or matching values)",
			ruleBackendLoc(ruleIdx, host, pathPattern), backend.Service.Mode, backend.Mode)
	}
	mode := svcMode
	if mode == "" {
		mode = bkMode
	}
	if mode == "" {
		mode = backendModeInternal
	}
	if mode != backendModeInternal && mode != backendModeExternal {
		return fmt.Errorf("%s: unsupported mode %q (use internal or external); set backend.service.mode or legacy backend.mode",
			ruleBackendLoc(ruleIdx, host, pathPattern), mode)
	}

	// backend.cache shape is shared by service / handler / redirect; validate as soon as enabled is true.
	if backend.Cache.Enabled {
		if err := validateBackendCache(backend.Cache, ruleIdx, host, pathPattern); err != nil {
			return err
		}
	}

	backendType := backend.Type
	if backendType == "" {
		backendType = backendTypeService
	}

	if backendType != backendTypeService && strings.TrimSpace(backend.Service.Mode) != "" {
		return fmt.Errorf("%s: backend.service.mode applies only to service upstreams",
			ruleBackendLoc(ruleIdx, host, pathPattern))
	}

	hr := hasRedirectBackend(backend)
	hs := servicePopulated(backend.Service)
	hh := handlerPopulated(backend.Handler)

	switch backendType {
	case backendTypeService:
		if hr {
			return fmt.Errorf("%s: backend.type is \"service\" but backend.redirect is set; use type \"redirect\" or remove redirect",
				ruleBackendLoc(ruleIdx, host, pathPattern))
		}
		if hh {
			return fmt.Errorf("%s: backend.type is \"service\" but backend.handler is configured; use type \"handler\" or remove handler",
				ruleBackendLoc(ruleIdx, host, pathPattern))
		}
		if err := backend.Service.Validate(); err != nil {
			return fmt.Errorf("%s.service: %w", ruleBackendLoc(ruleIdx, host, pathPattern), err)
		}
		if err := validateServiceAuth(backend.Service.Auth, ruleBackendLoc(ruleIdx, host, pathPattern)+".service"); err != nil {
			return err
		}
	case backendTypeRedirect:
		if !hr {
			return fmt.Errorf("%s: backend.type \"redirect\" requires backend.redirect.url",
				ruleBackendLoc(ruleIdx, host, pathPattern))
		}
		if hs {
			return fmt.Errorf("%s: backend.type \"redirect\" must not configure backend.service",
				ruleBackendLoc(ruleIdx, host, pathPattern))
		}
		if hh {
			return fmt.Errorf("%s: backend.type \"redirect\" must not configure backend.handler",
				ruleBackendLoc(ruleIdx, host, pathPattern))
		}
	case backendTypeHandler:
		if hr {
			return fmt.Errorf("%s: backend.type is \"handler\" but backend.redirect is set; use type \"redirect\" or remove redirect",
				ruleBackendLoc(ruleIdx, host, pathPattern))
		}
		if hs {
			return fmt.Errorf("%s: backend.type is \"handler\" but backend.service is configured; use type \"service\" or remove service",
				ruleBackendLoc(ruleIdx, host, pathPattern))
		}
		if err := validateHandler(backend.Handler, ruleIdx, host, pathPattern); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%s.type: unsupported backend type: %s",
			ruleBackendLoc(ruleIdx, host, pathPattern), backendType)
	}

	return nil
}

// validateBackendCache checks backend.cache field constraints (any backend type when enabled).
func validateBackendCache(cache rule.BackendCache, ruleIdx int, host, pathPattern string) error {
	loc := ruleBackendLoc(ruleIdx, host, pathPattern)
	switch strings.ToLower(strings.TrimSpace(cache.KeyHash)) {
	case "", httpCacheKeyHashMD5, httpCacheKeyHashSHA256:
	default:
		return fmt.Errorf("%s: backend.cache.key_hash must be md5 or sha256", loc)
	}
	if cache.TTL < 0 {
		return fmt.Errorf("%s: backend.cache.ttl must be >= 0", loc)
	}
	if cache.MaxBodyBytes < 0 {
		return fmt.Errorf("%s: backend.cache.max_body_bytes must be >= 0", loc)
	}
	for _, m := range cache.Methods {
		m = strings.ToUpper(strings.TrimSpace(m))
		if m == "" {
			return fmt.Errorf("%s: backend.cache.methods entries must be non-empty", loc)
		}
		if m == http.MethodPost {
			return fmt.Errorf("%s: backend.cache.methods must not include POST; use cache.paths[].methods and key_json", loc)
		}
	}
	for _, h := range cache.KeyHeaders {
		if strings.TrimSpace(h) == "" {
			return fmt.Errorf("%s: backend.cache.key_headers entries must be non-empty", loc)
		}
	}
	cache.KeyHeaders = normalizeHTTPCacheKeyHeaders(cache.KeyHeaders)
	for _, d := range cache.BypassRequestDirectives {
		if strings.TrimSpace(d) == "" {
			return fmt.Errorf("%s: backend.cache.bypass_request_directives entries must be non-empty", loc)
		}
	}
	switch strings.ToLower(strings.TrimSpace(cache.Default)) {
	case "", cachePathDefaultCache, cachePathDefaultBypass:
	default:
		return fmt.Errorf("%s: backend.cache.default must be cache or bypass", loc)
	}
	for i, pr := range cache.Paths {
		if strings.TrimSpace(pr.Match) == "" {
			return fmt.Errorf("%s: backend.cache.paths[%d].match must be non-empty", loc, i)
		}
		mt := strings.ToLower(strings.TrimSpace(pr.MatchType))
		switch mt {
		case "", cachePathMatchAuto, cachePathMatchPrefix, cachePathMatchExact, cachePathMatchRegex:
		default:
			return fmt.Errorf("%s: backend.cache.paths[%d].match_type must be auto, prefix, exact, or regex", loc, i)
		}
		action := strings.ToLower(strings.TrimSpace(pr.Action))
		if action == "" {
			action = cachePathActionCache
		}
		switch action {
		case cachePathActionCache, cachePathActionBypass:
		default:
			return fmt.Errorf("%s: backend.cache.paths[%d].action must be cache or bypass", loc, i)
		}
		if pr.TTL < 0 {
			return fmt.Errorf("%s: backend.cache.paths[%d].ttl must be >= 0", loc, i)
		}
		if pr.MaxBodyBytes < 0 {
			return fmt.Errorf("%s: backend.cache.paths[%d].max_body_bytes must be >= 0", loc, i)
		}
		if pr.KeyBodyMaxBytes < 0 {
			return fmt.Errorf("%s: backend.cache.paths[%d].key_body_max_bytes must be >= 0", loc, i)
		}
		if pr.KeyBodyMaxBytes > maxHTTPCacheKeyBodyMaxBytes {
			return fmt.Errorf("%s: backend.cache.paths[%d].key_body_max_bytes must be <= %d", loc, i, maxHTTPCacheKeyBodyMaxBytes)
		}
		keyJSON := make([]string, 0, len(pr.KeyJSON))
		for j, p := range pr.KeyJSON {
			p = strings.TrimSpace(p)
			if p == "" {
				return fmt.Errorf("%s: backend.cache.paths[%d].key_json[%d] must be non-empty", loc, i, j)
			}
			if err := validateKeyJSONDotPath(p); err != nil {
				return fmt.Errorf("%s: backend.cache.paths[%d].key_json[%d]: %w", loc, i, j, err)
			}
			keyJSON = append(keyJSON, p)
		}
		if len(keyJSON) > 0 && action != cachePathActionCache {
			return fmt.Errorf("%s: backend.cache.paths[%d].key_json requires action cache", loc, i)
		}
		effMethods := effectiveCacheMethodsForPathRule(pr, cache.Methods)
		if len(keyJSON) > 0 && !methodSetIncludes(effMethods, http.MethodPost) {
			return fmt.Errorf("%s: backend.cache.paths[%d].key_json requires paths[%d].methods to include POST", loc, i, i)
		}
		pathMethods := normalizeCacheMethodsList(pr.Methods)
		if len(pathMethods) > 0 && len(keyJSON) == 0 {
			postOnly := true
			for _, m := range pathMethods {
				if m != http.MethodPost {
					postOnly = false
					break
				}
			}
			if postOnly {
				return fmt.Errorf("%s: backend.cache.paths[%d].methods includes POST but key_json is empty", loc, i)
			}
		}
	}
	if err := compileBackendCachePathRules(&cache); err != nil {
		return fmt.Errorf("%s: %w", loc, err)
	}
	return nil
}

func ruleBackendLoc(ruleIdx int, host, pathPattern string) string {
	if ruleIdx < 0 {
		return fmt.Sprintf("fallback path=%q", pathPattern)
	}
	return fmt.Sprintf("rules[%d] host=%q path=%q", ruleIdx, host, pathPattern)
}

func handlerLoc(ruleIdx int, host, pathPattern string) string {
	return ruleBackendLoc(ruleIdx, host, pathPattern) + " handler"
}

func validateHandler(handler rule.Handler, ruleIdx int, host, pathPattern string) error {
	handlerType := handler.Type
	if handlerType == "" {
		handlerType = handlerTypeStaticResponse
	}

	switch handlerType {
	case handlerTypeStaticResponse:
		return nil
	case handlerTypeFileServer, handlerTypeTemplates:
		if handler.RootDir == "" {
			return fmt.Errorf("%s.root_dir is required for %s",
				handlerLoc(ruleIdx, host, pathPattern), handlerType)
		}
		return nil
	case handlerTypeScript:
		engine := handler.Engine
		if engine == "" {
			engine = scriptEngineJavaScript
		}

		switch engine {
		case scriptEngineJavaScript, scriptEngineGo:
			return nil
		default:
			return fmt.Errorf("%s.engine: unsupported script engine: %s",
				handlerLoc(ruleIdx, host, pathPattern), engine)
		}
	default:
		return fmt.Errorf("%s.type: unsupported handler type: %s",
			handlerLoc(ruleIdx, host, pathPattern), handlerType)
	}
}
