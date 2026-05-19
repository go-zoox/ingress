package core

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-zoox/ingress/core/rule"
)

// applyStripPrefix expands paths[].backend.service.strip_prefix: true into
// request.path.rewrites using the sibling paths[].path pattern.
func applyStripPrefix(cfg *Config) error {
	for i := range cfg.Rules {
		r := &cfg.Rules[i]
		if err := validateStripPrefixBackend(r.Backend, i, r.Host, "/"); err != nil {
			return err
		}
		for j := range r.Paths {
			p := &r.Paths[j]
			pathPat := p.Path
			if pathPat == "" {
				pathPat = fmt.Sprintf("paths[%d]", j)
			}
			if err := applyPathStripPrefix(p, i, r.Host, pathPat); err != nil {
				return err
			}
		}
	}
	if err := validateStripPrefixBackend(cfg.Fallback, -1, "", "/"); err != nil {
		return err
	}
	return nil
}

func validateStripPrefixBackend(backend rule.Backend, ruleIdx int, host, pathPattern string) error {
	if !backend.Service.StripPrefix {
		return nil
	}
	return fmt.Errorf("%s: strip_prefix is only valid on paths[].backend.service (not rule-level or fallback backends)",
		ruleBackendLoc(ruleIdx, host, pathPattern))
}

func applyPathStripPrefix(p *rule.Path, ruleIdx int, host, pathPat string) error {
	svc := &p.Backend.Service
	if !svc.StripPrefix {
		return nil
	}
	loc := ruleBackendLoc(ruleIdx, host, pathPat)
	if strings.TrimSpace(p.Path) == "" {
		return fmt.Errorf("%s: strip_prefix requires a non-empty paths[].path", loc)
	}
	if len(svc.Request.Path.Rewrites) > 0 {
		return fmt.Errorf("%s: strip_prefix cannot be used together with request.path.rewrites", loc)
	}
	rw, err := stripPrefixRewriteRule(p.Path)
	if err != nil {
		return fmt.Errorf("%s: %w", loc, err)
	}
	svc.Request.Path.Rewrites = []string{rw}
	svc.StripPrefix = false
	return nil
}

// stripPrefixRewriteRule builds a pattern:replacement rewrite that strips the matched path prefix.
// pathPattern uses the same syntax as paths[].path (anchored with ^ at match time).
func stripPrefixRewriteRule(pathPattern string) (string, error) {
	if _, err := regexp.Compile("^" + pathPattern); err != nil {
		return "", fmt.Errorf("invalid paths[].path for strip_prefix: %w", err)
	}

	var from, to string
	if strings.HasSuffix(pathPattern, "(.*)") {
		from = "^" + pathPattern
		to = "/$1"
	} else {
		from = "^" + pathPattern + "/?(.*)"
		to = "/$1"
	}
	if _, err := regexp.Compile(from); err != nil {
		return "", fmt.Errorf("strip_prefix rewrite pattern: %w", err)
	}
	return from + ":" + to, nil
}
