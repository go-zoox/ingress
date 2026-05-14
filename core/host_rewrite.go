package core

import (
	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/ingress/core/service"
)

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
	mode := bk.Mode
	if mode == "" {
		mode = backendModeInternal
	}
	if mode == backendModeExternal {
		return true
	}
	if matchedRule != nil && matchedRule.Host == fallbackRuleHost {
		return true
	}
	return false
}
