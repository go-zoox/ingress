package core

import (
	"fmt"
	"strconv"

	"github.com/go-zoox/ingress/core/rule"
)

// ValidateConfig performs static configuration checks without starting servers
// or touching external systems.
func ValidateConfig(cfg *Config) error {
	if _, err := compileRouterIndex(cfg.Rules, cfg.Fallback); err != nil {
		return fmt.Errorf("router rules: %w", err)
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

	return nil
}

// validateBackend checks one backend under a rule.
// host is the rule's host pattern; pathPattern is paths[].path from config for path backends,
// "/" for the rule-level backend. If paths[].path is empty, messages use paths[index] as fallback.
func validateBackend(backend rule.Backend, ruleIdx int, host, pathPattern string) error {
	backendType := backend.Type
	if backendType == "" {
		backendType = backendTypeService
	}

	switch backendType {
	case backendTypeService:
		if backend.Redirect.URL != "" && backend.Service.Name != "" {
			return fmt.Errorf("%s: backend.redirect and backend.service are mutually exclusive",
				ruleBackendLoc(ruleIdx, host, pathPattern))
		}
		if backend.Redirect.URL != "" {
			return nil
		}
		if err := backend.Service.Validate(); err != nil {
			return fmt.Errorf("%s.service: %w", ruleBackendLoc(ruleIdx, host, pathPattern), err)
		}
	case backendTypeHandler:
		if err := validateHandler(backend.Handler, ruleIdx, host, pathPattern); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%s.type: unsupported backend type: %s",
			ruleBackendLoc(ruleIdx, host, pathPattern), backendType)
	}

	return nil
}

func ruleBackendLoc(ruleIdx int, host, pathPattern string) string {
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
