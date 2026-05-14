package core

import (
	"fmt"
	"strings"

	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/ingress/core/service"
)

func hasRedirectBackend(b rule.Backend) bool {
	return strings.TrimSpace(b.Redirect.URL) != ""
}

// servicePopulated reports whether backend.service carries non-default configuration,
// used to infer backend.type or detect conflicts when type is explicit.
func servicePopulated(s service.Service) bool {
	if strings.TrimSpace(s.Name) != "" {
		return true
	}
	if s.Port != 0 {
		return true
	}
	if pt := strings.ToLower(strings.TrimSpace(s.Protocol)); pt != "" && pt != schemeHTTP {
		return true
	}
	if s.Auth.Type != "" {
		return true
	}
	if s.HealthCheck.Enable || s.HealthCheck.Ok {
		return true
	}
	if s.Request.Host.Rewrite {
		return true
	}
	if len(s.Request.Path.Rewrites) > 0 || len(s.Request.Headers) > 0 || len(s.Request.Query) > 0 {
		return true
	}
	if s.Request.Delay != 0 || s.Request.Timeout != 0 {
		return true
	}
	if len(s.Response.Headers) > 0 {
		return true
	}
	return false
}

// handlerPopulated reports whether backend.handler carries non-default configuration.
func handlerPopulated(h rule.Handler) bool {
	if strings.TrimSpace(h.Script) != "" || strings.TrimSpace(h.RootDir) != "" {
		return true
	}
	if len(h.Headers) > 0 {
		return true
	}
	if strings.TrimSpace(h.Body) != "" {
		return true
	}
	if h.Type != "" && h.Type != handlerTypeStaticResponse {
		return true
	}
	if h.Engine != "" && h.Engine != scriptEngineJavaScript {
		return true
	}
	if h.StatusCode != 0 && h.StatusCode != 200 {
		return true
	}
	if strings.TrimSpace(h.IndexFile) != "" && h.IndexFile != "index.html" {
		return true
	}
	return false
}

// inferRuleBackends sets Backend.Type when omitted and unambiguous (exactly one of service /
// handler / redirect signals). If two or more signals are present with Type empty, it errors.
// Pure service / redirect / handler blocks each produce exactly one signal.
func inferRuleBackends(rules []rule.Rule) error {
	for i := range rules {
		if err := inferOneBackend(&rules[i].Backend, i, rules[i].Host, "/"); err != nil {
			return err
		}
		for j := range rules[i].Paths {
			pathPat := rules[i].Paths[j].Path
			if pathPat == "" {
				pathPat = fmt.Sprintf("paths[%d]", j)
			}
			if err := inferOneBackend(&rules[i].Paths[j].Backend, i, rules[i].Host, pathPat); err != nil {
				return err
			}
		}
	}
	return nil
}

func inferOneBackend(b *rule.Backend, ruleIdx int, host, pathPattern string) error {
	if strings.TrimSpace(b.Type) != "" {
		return nil
	}
	hr := hasRedirectBackend(*b)
	hs := servicePopulated(b.Service)
	hh := handlerPopulated(b.Handler)

	n := 0
	if hr {
		n++
	}
	if hs {
		n++
	}
	if hh {
		n++
	}
	if n > 1 {
		return fmt.Errorf("%s: ambiguous backend: configure only one of backend.service, backend.handler, or backend.redirect, or set backend.type to \"service\", \"handler\", or \"redirect\" explicitly",
			ruleBackendLoc(ruleIdx, host, pathPattern))
	}
	switch n {
	case 1:
		switch {
		case hr:
			b.Type = backendTypeRedirect
		case hh:
			b.Type = backendTypeHandler
		default:
			b.Type = backendTypeService
		}
	default:
		// leave empty → validate treats as service default
	}
	return nil
}

func inferBackendTypes(cfg *Config) error {
	return inferRuleBackends(cfg.Rules)
}

// inferPathSliceBackends applies the same inference rules to path-backend slices (e.g. MatchPath tests).
func inferPathSliceBackends(paths []rule.Path, ruleIdx int, ruleHost string) error {
	for j := range paths {
		pathPat := paths[j].Path
		if pathPat == "" {
			pathPat = fmt.Sprintf("paths[%d]", j)
		}
		if err := inferOneBackend(&paths[j].Backend, ruleIdx, ruleHost, pathPat); err != nil {
			return err
		}
	}
	return nil
}
