package core

import (
	"testing"

	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/ingress/core/service"
)

func TestEffectiveHostRewrite_explicitTrue(t *testing.T) {
	tb := true
	s := &service.Service{Request: service.Request{Host: service.RequestHost{Rewrite: &tb}}}
	if !effectiveHostRewrite(s, nil, &rule.Rule{Backend: rule.Backend{Mode: backendModeInternal}}) {
		t.Fatal("expected true")
	}
}

func TestEffectiveHostRewrite_explicitFalseOverridesExternal(t *testing.T) {
	fb := false
	s := &service.Service{Request: service.Request{Host: service.RequestHost{Rewrite: &fb}}}
	if effectiveHostRewrite(s, nil, &rule.Rule{Backend: rule.Backend{Mode: backendModeExternal}}) {
		t.Fatal("expected false")
	}
}

func TestEffectiveHostRewrite_externalDefault(t *testing.T) {
	s := &service.Service{}
	if !effectiveHostRewrite(s, nil, &rule.Rule{Backend: rule.Backend{Mode: backendModeExternal}}) {
		t.Fatal("expected true")
	}
}

func TestEffectiveHostRewrite_internalDefault(t *testing.T) {
	s := &service.Service{}
	if effectiveHostRewrite(s, nil, &rule.Rule{Backend: rule.Backend{Mode: backendModeInternal}}) {
		t.Fatal("expected false")
	}
}

func TestEffectiveHostRewrite_fallbackRoute(t *testing.T) {
	s := &service.Service{}
	if !effectiveHostRewrite(s, nil, &rule.Rule{Host: fallbackRuleHost, Backend: rule.Backend{Mode: backendModeInternal}}) {
		t.Fatal("expected true for fallback host when rewrite unset")
	}
}

func TestEffectiveHostRewrite_pathBackendExternal(t *testing.T) {
	s := &service.Service{}
	pathBk := &rule.Backend{Mode: backendModeExternal}
	hostBk := rule.Backend{Mode: backendModeInternal}
	if !effectiveHostRewrite(s, pathBk, &rule.Rule{Backend: hostBk}) {
		t.Fatal("expected path backend mode to win")
	}
}
