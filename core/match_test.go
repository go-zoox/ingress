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

	s, err := Match(rules, "portainer.example.com", "/")
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "portainer" {
		t.Fatalf("expected portainer, got %s", s.Name)
	}
	if s.Port != 8080 {
		t.Fatalf("expected 8080, got %d", s.Port)
	}
	if s.Protocol != "http" {
		t.Fatalf("expected http, got %s", s.Protocol)
	}

	s, err = Match(rules, "docker-registry.example.com", "/")
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "docker-registry" {
		t.Fatalf("expected docker-registry, got %s", s.Name)
	}
	if s.Port != 8080 {
		t.Fatalf("expected 8080, got %d", s.Port)
	}
	if s.Protocol != "http" {
		t.Fatalf("expected http, got %s", s.Protocol)
	}

	s, err = Match(rules, "docker-registry.example.work", "/")
	if err != nil {
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

	s, err := Match(rules, "httpbin.example.work", "/ip")
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "httpbin.zcorky.com" {
		t.Fatalf("expected httpbin.zcorky.com, got %s", s.Name)
	}
	if s.Port != 443 {
		t.Fatalf("expected 443, got %d", s.Port)
	}
	if s.Protocol != "https" {
		t.Fatalf("expected https, got %s", s.Protocol)
	}

	s, err = Match(rules, "httpbin.example.work", "/ip1")
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

	s, err = Match(rules, "httpbin.example.work", "/ip2")
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
}

func TestMatchHostRewriteName(t *testing.T) {
	rules := []rule.Rule{
		{
			Host: "t-(\\w+).example.work",
			Backend: rule.Backend{
				Service: service.Service{
					Protocol: "http",
					Name:     "task.$1.svc",
					Port:     8080,
				},
			},
		},
	}

	s, err := Match(rules, "t-zero.example.work", "/")
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "task.zero.svc" {
		t.Fatalf("expected portainer, got %s", s.Name)
	}
	if s.Port != 8080 {
		t.Fatalf("expected 8080, got %d", s.Port)
	}
	if s.Protocol != "http" {
		t.Fatalf("expected http, got %s", s.Protocol)
	}
}
