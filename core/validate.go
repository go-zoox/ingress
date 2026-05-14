package core

import (
	"fmt"
	"strconv"
	"strings"

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

	if strings.TrimSpace(cfg.Fallback.Service.Name) != "" {
		if err := validateBackend(cfg.Fallback, -1, "", "/"); err != nil {
			return err
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
	mode := strings.TrimSpace(backend.Mode)
	if mode == "" {
		mode = backendModeInternal
	}
	if mode != backendModeInternal && mode != backendModeExternal {
		return fmt.Errorf("%s: unsupported backend.mode %q (use internal or external)",
			ruleBackendLoc(ruleIdx, host, pathPattern), backend.Mode)
	}

	backendType := backend.Type
	if backendType == "" {
		backendType = backendTypeService
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
