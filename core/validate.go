package core

import (
	"fmt"

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
		if err := validateBackend(cfg.Rules[i].Backend, fmt.Sprintf("rules[%d].backend", i)); err != nil {
			return err
		}

		for j := range cfg.Rules[i].Paths {
			if err := validateBackend(cfg.Rules[i].Paths[j].Backend, fmt.Sprintf("rules[%d].paths[%d].backend", i, j)); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateBackend(backend rule.Backend, path string) error {
	backendType := backend.Type
	if backendType == "" {
		backendType = backendTypeService
	}

	switch backendType {
	case backendTypeService:
		if err := backend.Service.Validate(); err != nil {
			return fmt.Errorf("%s.service: %w", path, err)
		}
	case backendTypeHandler:
		if err := validateHandler(backend.Handler, path+".handler"); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%s.type: unsupported backend type: %s", path, backendType)
	}

	return nil
}

func validateHandler(handler rule.Handler, path string) error {
	handlerType := handler.Type
	if handlerType == "" {
		handlerType = handlerTypeStaticResponse
	}

	switch handlerType {
	case handlerTypeStaticResponse:
		return nil
	case handlerTypeFileServer, handlerTypeTemplates:
		if handler.RootDir == "" {
			return fmt.Errorf("%s.root_dir is required for %s", path, handlerType)
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
			return fmt.Errorf("%s.engine: unsupported script engine: %s", path, engine)
		}
	default:
		return fmt.Errorf("%s.type: unsupported handler type: %s", path, handlerType)
	}
}
