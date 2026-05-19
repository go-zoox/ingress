package core

import (
	"strings"

	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/ingress/core/service"
)

// effectiveBackendMode returns internal or external for Host rewrite. Prefer
// backend.service.mode; if empty, use legacy backend.mode.
func effectiveBackendMode(bk rule.Backend) string {
	svc := strings.TrimSpace(bk.Service.Mode)
	bkM := strings.TrimSpace(bk.Mode)
	if svc != "" {
		return svc
	}
	if bkM != "" {
		return bkM
	}
	return backendModeInternal
}

func effectiveBackendForHostRewrite(pathBackend *rule.Backend, matchedRule *rule.Rule) rule.Backend {
	if pathBackend != nil && getBackendType(*pathBackend) == backendTypeService {
		return *pathBackend
	}
	if matchedRule == nil {
		return rule.Backend{}
	}
	return matchedRule.Backend
}

// effectiveHostRewrite decides whether the outbound Host header is set to the upstream (service) host.
func effectiveHostRewrite(s *service.Service, pathBackend *rule.Backend, matchedRule *rule.Rule) bool {
	if s.Request.Host.Rewrite != nil {
		return *s.Request.Host.Rewrite
	}
	bk := effectiveBackendForHostRewrite(pathBackend, matchedRule)
	mode := effectiveBackendMode(bk)
	if mode == backendModeExternal {
		return true
	}
	if matchedRule != nil && matchedRule.Host == fallbackRuleHost {
		return true
	}
	return false
}
