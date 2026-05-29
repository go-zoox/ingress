package core

import (
	"testing"

	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/ingress/core/service"
)

func TestMatchHost(t *testing.T) {
	rules := []rule.Rule{
		{
			Host: "portainer.example.com",
			Backend: rule.Backend{
				Service: service.Service{
					Protocol: "http",
					Name:     "portainer",
					Port:     8080,
				},
			},
		},
		{
			Host: "docker-registry.example.com",
			Backend: rule.Backend{
				Service: service.Service{
					Protocol: "http",
					Name:     "docker-registry",
					Port:     8080,
				},
			},
		},
	}

	s, err := MatchHost(rules, rule.Backend{}, "portainer.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if s.Service.Name != "portainer" {
		t.Fatalf("expected portainer, got %s", s.Service.Name)
	}
	if s.Service.Port != 8080 {
		t.Fatalf("expected 8080, got %d", s.Service.Port)
	}
	if s.Service.Protocol != "http" {
		t.Fatalf("expected http, got %s", s.Service.Protocol)
	}

	s, err = MatchHost(rules, rule.Backend{}, "docker-registry.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if s.Service.Name != "docker-registry" {
		t.Fatalf("expected docker-registry, got %s", s.Service.Name)
	}
	if s.Service.Port != 8080 {
		t.Fatalf("expected 8080, got %d", s.Service.Port)
	}
	if s.Service.Protocol != "http" {
		t.Fatalf("expected http, got %s", s.Service.Protocol)
	}

	s, err = MatchHost(rules, rule.Backend{}, "docker-registry.example.work")
	if err == nil {
		t.Fatal(err)
	}
	if s != nil {
		t.Fatalf("expected nil, got %v", s)
	}
}

func TestMatchPath(t *testing.T) {
	rules := []rule.Rule{
		{
			Host: "portainer.example.com",
			Backend: rule.Backend{
				Service: service.Service{
					Protocol: "http",
					Name:     "portainer",
					Port:     8080,
				},
			},
		},
		{
			Host: "docker-registry.example.com",
			Backend: rule.Backend{
				Service: service.Service{
					Protocol: "http",
					Name:     "docker-registry",
					Port:     8080,
				},
			},
			Paths: []rule.Path{
				{
					Path: "/v2",
					Backend: rule.Backend{
						Service: service.Service{
							Protocol: "http",
							Name:     "docker-registry-v2",
							Port:     8080,
						},
					},
				},
			},
		},
		{
			Host: "httpbin.example.work",
			Backend: rule.Backend{
				Service: service.Service{
					Protocol: "https",
					Name:     "httpbin.zcorky.com",
					Port:     443,
				},
			},
			Paths: []rule.Path{
				{
					Path: "/ip1",
					Backend: rule.Backend{
						Service: service.Service{
							Protocol: "http",
							Name:     "ip3.httpbin.zcorky.com",
							Port:     443,
						},
					},
				},
				{
					Path: "/ip2",
					Backend: rule.Backend{
						Service: service.Service{
							Protocol: "https",
							Name:     "ip2.httpbin.zcorky.com",
							Port:     443,
						},
					},
				},
			},
		},
	}

	s, matchedPath, err := MatchPath(rules[2].Paths, "/ip")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if s != nil {
		t.Fatalf("expected nil, got %v", s)
	}
	if matchedPath != nil {
		t.Fatalf("expected nil matchedPath, got %v", matchedPath)
	}

	s, matchedPath, err = MatchPath(rules[2].Paths, "/ip1")
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "ip3.httpbin.zcorky.com" {
		t.Fatalf("expected ip3.httpbin.zcorky.com, got %s", s.Name)
	}
	if s.Port != 443 {
		t.Fatalf("expected 443, got %d", s.Port)
	}
	if s.Protocol != "http" {
		t.Fatalf("expected http, got %s", s.Protocol)
	}
	if matchedPath == nil {
		t.Fatal("expected matchedPath, got nil")
	}
	if matchedPath.Path != "/ip1" {
		t.Fatalf("expected /ip1, got %s", matchedPath.Path)
	}

	s, matchedPath, err = MatchPath(rules[2].Paths, "/ip2")
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "ip2.httpbin.zcorky.com" {
		t.Fatalf("expected ip2.httpbin.zcorky.com, got %s", s.Name)
	}
	if s.Port != 443 {
		t.Fatalf("expected 443, got %d", s.Port)
	}
	if s.Protocol != "https" {
		t.Fatalf("expected https, got %s", s.Protocol)
	}
	if matchedPath == nil {
		t.Fatal("expected matchedPath, got nil")
	}
	if matchedPath.Path != "/ip2" {
		t.Fatalf("expected /ip2, got %s", matchedPath.Path)
	}
}

func TestMatchHostRewriteName(t *testing.T) {
	rules := []rule.Rule{
		{
			Host:     "^t-(\\w+).example.work",
			HostType: "regex",
			Backend: rule.Backend{
				Service: service.Service{
					Protocol: "http",
					Name:     "task.$1.svc",
					Port:     8080,
				},
			},
		},
	}

	s, err := MatchHost(rules, rule.Backend{}, "t-zero.example.work")
	if err != nil {
		t.Fatal(err)
	}
	if s.Service.Name != "task.zero.svc" {
		t.Fatalf("expected portainer, got %s", s.Service.Name)
	}
	if s.Service.Port != 8080 {
		t.Fatalf("expected 8080, got %d", s.Service.Port)
	}
	if s.Service.Protocol != "http" {
		t.Fatalf("expected http, got %s", s.Service.Protocol)
	}
}

func TestMatchHostRewriteNameForPathBackend(t *testing.T) {
	rules := []rule.Rule{
		{
			Host:     "^t-(\\w+).example.work",
			HostType: "regex",
			Backend: rule.Backend{
				Service: service.Service{
					Protocol: "http",
					Name:     "task.$1.svc",
					Port:     8080,
				},
			},
			Paths: []rule.Path{
				{
					Path: "/api/v1/[^/]+",
					Backend: rule.Backend{
						Service: service.Service{
							Protocol: "http",
							Name:     "$1.example.work",
							Port:     8080,
						},
					},
				},
			},
		},
	}

	idx, err := compileRouterIndex(rules, rule.Backend{})
	if err != nil {
		t.Fatal(err)
	}

	s, matchedPath, _, _, err := matchPathWithRouter(idx, rules, 0, "/api/v1/demo", "t-zero.example.work", nil)
	if err != nil {
		t.Fatal(err)
	}
	if matchedPath == nil {
		t.Fatal("expected matchedPath, got nil")
	}
	if s == nil {
		t.Fatal("expected service, got nil")
	}
	if s.Name != "zero.example.work" {
		t.Fatalf("expected zero.example.work, got %s", s.Name)
	}
}

func TestMatchServiceNameTemplateWithHostAndPathCaptures(t *testing.T) {
	rules := []rule.Rule{
		{
			Host:     "^t-(\\w+).example.work$",
			HostType: "regex",
			Backend: rule.Backend{
				Service: service.Service{
					Protocol: "http",
					Name:     "task.${host.1}.svc",
					Port:     8080,
				},
			},
			Paths: []rule.Path{
				{
					Path: "^/api/v1/([^/]+)$",
					Backend: rule.Backend{
						Service: service.Service{
							Protocol: "http",
							Name:     "${path.1}.${host.1}.svc",
							Port:     8080,
						},
					},
				},
			},
		},
	}

	hm, err := MatchHost(rules, rule.Backend{}, "t-zero.example.work")
	if err != nil {
		t.Fatal(err)
	}
	if hm.Service == nil {
		t.Fatal("expected host service, got nil")
	}
	if hm.Service.Name != "task.zero.svc" {
		t.Fatalf("expected task.zero.svc, got %s", hm.Service.Name)
	}

	idx, err := compileRouterIndex(rules, rule.Backend{})
	if err != nil {
		t.Fatal(err)
	}
	s, matchedPath, _, _, err := matchPathWithRouter(idx, rules, 0, "/api/v1/order", "t-zero.example.work", hm.hostSubmatches)
	if err != nil {
		t.Fatal(err)
	}
	if matchedPath == nil {
		t.Fatal("expected matchedPath, got nil")
	}
	if s == nil {
		t.Fatal("expected service, got nil")
	}
	if s.Name != "order.zero.svc" {
		t.Fatalf("expected order.zero.svc, got %s", s.Name)
	}
}

func TestMatchServiceNameTemplateWithIndexedCaptures(t *testing.T) {
	rules := []rule.Rule{
		{
			Host:     "^t-(\\w+)-(dev|prod).example.work$",
			HostType: "regex",
			Backend: rule.Backend{
				Service: service.Service{
					Protocol: "http",
					Name:     "${host.2}-${host.1}.svc",
					Port:     8080,
				},
			},
			Paths: []rule.Path{
				{
					Path: "^/api/v1/([^/]+)/([^/]+)$",
					Backend: rule.Backend{
						Service: service.Service{
							Protocol: "http",
							Name:     "${path.2}.${path.1}.${host.2}.${host.1}.svc",
							Port:     8080,
						},
					},
				},
				{
					Path: "^/api/v2/([^/]+)$",
					Backend: rule.Backend{
						Service: service.Service{
							Protocol: "http",
							Name:     "${path.9}.${host.9}.svc",
							Port:     8080,
						},
					},
				},
			},
		},
	}

	hm, err := MatchHost(rules, rule.Backend{}, "t-zero-dev.example.work")
	if err != nil {
		t.Fatal(err)
	}
	if hm.Service == nil {
		t.Fatal("expected host service, got nil")
	}
	if hm.Service.Name != "dev-zero.svc" {
		t.Fatalf("expected dev-zero.svc, got %s", hm.Service.Name)
	}

	idx, err := compileRouterIndex(rules, rule.Backend{})
	if err != nil {
		t.Fatal(err)
	}

	s, matchedPath, _, _, err := matchPathWithRouter(idx, rules, 0, "/api/v1/order/create", "t-zero-dev.example.work", hm.hostSubmatches)
	if err != nil {
		t.Fatal(err)
	}
	if matchedPath == nil {
		t.Fatal("expected matchedPath, got nil")
	}
	if s == nil {
		t.Fatal("expected service, got nil")
	}
	if s.Name != "create.order.dev.zero.svc" {
		t.Fatalf("expected create.order.dev.zero.svc, got %s", s.Name)
	}

	s, _, _, _, err = matchPathWithRouter(idx, rules, 0, "/api/v2/onlyone", "t-zero-dev.example.work", hm.hostSubmatches)
	if err != nil {
		t.Fatal(err)
	}
	if s == nil {
		t.Fatal("expected service, got nil")
	}
	if s.Name != "${path.9}.${host.9}.svc" {
		t.Fatalf("expected unresolved placeholders, got %s", s.Name)
	}
}

func TestExpandRedirectURL_RegexHostCaptures(t *testing.T) {
	rules := []rule.Rule{
		{
			Host:     `^bigscreen-([^.]+)\.ys\.example\.com$`,
			HostType: "regex",
			Backend: rule.Backend{
				Service: service.Service{
					Protocol: "http",
					Name:     "noop",
					Port:     8080,
				},
			},
		},
	}

	hm, err := MatchHost(rules, rule.Backend{}, "bigscreen-acme.ys.example.com")
	if err != nil {
		t.Fatal(err)
	}

	host := "bigscreen-acme.ys.example.com"
	got := expandRedirectURL(hm.Rule, host, "https://bigscreen-$1.yss.example.com", hm.hostSubmatches, nil)
	if got != "https://bigscreen-acme.yss.example.com" {
		t.Fatalf("legacy $1: got %q", got)
	}
	got2 := expandRedirectURL(hm.Rule, host, "https://bigscreen-${host.1}.yss.example.com", hm.hostSubmatches, nil)
	if got2 != "https://bigscreen-acme.yss.example.com" {
		t.Fatalf("${host.1}: got %q", got2)
	}
}

func TestMatchHost_RedirectOnlyBackendNoService(t *testing.T) {
	rules := []rule.Rule{
		{
			Host: "redirect-host.example.com",
			Backend: rule.Backend{
				Redirect: rule.Redirect{
					URL: "https://target.example/",
				},
			},
		},
	}

	hm, err := MatchHost(rules, rule.Backend{}, "redirect-host.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if hm.Service != nil {
		t.Fatal("expected nil matcher Service for redirect backend")
	}
	if getBackendType(hm.Rule.Backend) != backendTypeRedirect {
		t.Fatalf("expected redirect backend type, got %q", hm.Rule.Backend.Type)
	}
}

func TestMatchHostWithFallback(t *testing.T) {
	rules := []rule.Rule{
		{
			Host: "portainer.example.com",
			Backend: rule.Backend{
				Service: service.Service{
					Protocol: "http",
					Name:     "portainer",
					Port:     8080,
				},
			},
		},
	}

	fallback := rule.Backend{
		Service: service.Service{
			Protocol: "http",
			Name:     "fallback",
			Port:     8080,
		},
	}

	s, err := MatchHost(rules, fallback, "portainer.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if s.Service.Name != "portainer" {
		t.Fatalf("expected portainer, got %s", s.Service.Name)
	}
	if s.Service.Port != 8080 {
		t.Fatalf("expected 8080, got %d", s.Service.Port)
	}
	if s.Service.Protocol != "http" {
		t.Fatalf("expected http, got %s", s.Service.Protocol)
	}

	s, err = MatchHost(rules, fallback, "portainer.example.work")
	if err != nil {
		t.Fatal(err)
	}
	if s.Service.Name != "fallback" {
		t.Fatalf("expected fallback, got %s", s.Service.Name)
	}
	if s.Service.Port != 8080 {
		t.Fatalf("expected 8080, got %d", s.Service.Port)
	}
	if s.Service.Protocol != "http" {
		t.Fatalf("expected http, got %s", s.Service.Protocol)
	}
}

func TestMatchHostWithHandlerBackend(t *testing.T) {
	rules := []rule.Rule{
		{
			Host: "handler.example.work",
			Backend: rule.Backend{
				Type: backendTypeHandler,
				Handler: rule.Handler{
					Body: "Hello World!",
				},
			},
		},
	}

	s, err := MatchHost(rules, rule.Backend{}, "handler.example.work")
	if err != nil {
		t.Fatal(err)
	}
	if s == nil {
		t.Fatal("expected host matcher, got nil")
	}
	if s.Service != nil {
		t.Fatalf("expected nil service for handler backend, got %v", s.Service)
	}
}

func TestMatchPathWithHandlerBackend(t *testing.T) {
	paths := []rule.Path{
		{
			Path: "/custom/handler/string",
			Backend: rule.Backend{
				Type: backendTypeHandler,
				Handler: rule.Handler{
					Body: "Hello World!",
				},
			},
		},
	}

	s, matchedPath, err := MatchPath(paths, "/custom/handler/string")
	if err != nil {
		t.Fatal(err)
	}
	if s != nil {
		t.Fatalf("expected nil service for handler backend, got %v", s)
	}
	if matchedPath == nil {
		t.Fatal("expected matchedPath, got nil")
	}
	if matchedPath.Backend.Type != backendTypeHandler {
		t.Fatalf("expected handler backend type, got %s", matchedPath.Backend.Type)
	}
}

// TestMatchHost_AuthFieldPropagated is a regression test for the bug where
// service.Service was constructed in hostMatcherFromMatchedRule without copying
// the Auth field, causing ValidateAuth to always see an empty Auth and allow all
// requests regardless of the configured authentication.
func TestMatchHost_AuthFieldPropagated(t *testing.T) {
	rules := []rule.Rule{
		{
			Host: "basic-auth.example.com",
			Backend: rule.Backend{
				Service: service.Service{
					Protocol: "https",
					Name:     "upstream.example.com",
					Port:     443,
					Auth: service.Auth{
						Type: "basic",
						Basic: service.BasicAuth{
							Users: []service.BasicUser{
								{Username: "admin", Password: "admin123"},
								{Username: "user1", Password: "user123"},
							},
						},
					},
				},
			},
		},
	}

	hm, err := MatchHost(rules, rule.Backend{}, "basic-auth.example.com")
	if err != nil {
		t.Fatalf("MatchHost failed: %v", err)
	}

	if hm.Service.Auth.Type != "basic" {
		t.Fatalf("expected Auth.Type=\"basic\", got %q — Auth field not propagated from rule config", hm.Service.Auth.Type)
	}
	if len(hm.Service.Auth.Basic.Users) != 2 {
		t.Fatalf("expected 2 basic auth users, got %d", len(hm.Service.Auth.Basic.Users))
	}
	if hm.Service.Auth.Basic.Users[0].Username != "admin" {
		t.Fatalf("expected first user to be admin, got %q", hm.Service.Auth.Basic.Users[0].Username)
	}
}

// TestMatchHost_BearerAuthFieldPropagated ensures Bearer auth config is also preserved.
func TestMatchHost_BearerAuthFieldPropagated(t *testing.T) {
	rules := []rule.Rule{
		{
			Host: "bearer-auth.example.com",
			Backend: rule.Backend{
				Service: service.Service{
					Protocol: "http",
					Name:     "api-service",
					Port:     80,
					Auth: service.Auth{
						Type: "bearer",
						Bearer: service.BearerAuth{
							Tokens: []string{"secret-token-abc", "secret-token-xyz"},
						},
					},
				},
			},
		},
	}

	hm, err := MatchHost(rules, rule.Backend{}, "bearer-auth.example.com")
	if err != nil {
		t.Fatalf("MatchHost failed: %v", err)
	}

	if hm.Service.Auth.Type != "bearer" {
		t.Fatalf("expected Auth.Type=\"bearer\", got %q", hm.Service.Auth.Type)
	}
	if len(hm.Service.Auth.Bearer.Tokens) != 2 {
		t.Fatalf("expected 2 bearer tokens, got %d", len(hm.Service.Auth.Bearer.Tokens))
	}
}

// TestMatchHost_RegexHost_AuthFieldPropagated ensures the Auth field is copied even
// when the host is matched via a regex rule (the second branch in hostMatcherFromMatchedRule).
func TestMatchHost_RegexHost_AuthFieldPropagated(t *testing.T) {
	rules := []rule.Rule{
		{
			Host:     `^secure-([a-z]+)\.example\.com$`,
			HostType: "regex",
			Backend: rule.Backend{
				Service: service.Service{
					Protocol: "http",
					Name:     "$1.internal",
					Port:     80,
					Auth: service.Auth{
						Type: "basic",
						Basic: service.BasicAuth{
							Users: []service.BasicUser{
								{Username: "ops", Password: "opspass"},
							},
						},
					},
				},
			},
		},
	}

	hm, err := MatchHost(rules, rule.Backend{}, "secure-frontend.example.com")
	if err != nil {
		t.Fatalf("MatchHost failed: %v", err)
	}

	if hm.Service.Auth.Type != "basic" {
		t.Fatalf("expected Auth.Type=\"basic\" for regex host match, got %q", hm.Service.Auth.Type)
	}
	if len(hm.Service.Auth.Basic.Users) != 1 || hm.Service.Auth.Basic.Users[0].Username != "ops" {
		t.Fatalf("expected user ops, got %v", hm.Service.Auth.Basic.Users)
	}
}
