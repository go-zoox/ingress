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
					Host:     "portainer",
					Port:     8080,
				},
			},
		},
		{
			Host: "docker-registry.example.com",
			Backend: rule.Backend{
				Service: service.Service{
					Protocol: "http",
					Host:     "docker-registry",
					Port:     8080,
				},
			},
		},
	}

	s, err := MatchHost(rules, "portainer.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if s.Service.Host != "portainer" {
		t.Fatalf("expected portainer, got %s", s.Service.Host)
	}
	if s.Service.Port != 8080 {
		t.Fatalf("expected 8080, got %d", s.Service.Port)
	}
	if s.Service.Protocol != "http" {
		t.Fatalf("expected http, got %s", s.Service.Protocol)
	}

	s, err = MatchHost(rules, "docker-registry.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if s.Service.Host != "docker-registry" {
		t.Fatalf("expected docker-registry, got %s", s.Service.Host)
	}
	if s.Service.Port != 8080 {
		t.Fatalf("expected 8080, got %d", s.Service.Port)
	}
	if s.Service.Protocol != "http" {
		t.Fatalf("expected http, got %s", s.Service.Protocol)
	}

	s, err = MatchHost(rules, "docker-registry.example.work")
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
					Host:     "portainer",
					Port:     8080,
				},
			},
		},
		{
			Host: "docker-registry.example.com",
			Backend: rule.Backend{
				Service: service.Service{
					Protocol: "http",
					Host:     "docker-registry",
					Port:     8080,
				},
			},
			Paths: []rule.Path{
				{
					Path: "/v2",
					Backend: rule.Backend{
						Service: service.Service{
							Protocol: "http",
							Host:     "docker-registry-v2",
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
					Host:     "httpbin.zcorky.com",
					Port:     443,
				},
			},
			Paths: []rule.Path{
				{
					Path: "/ip1",
					Backend: rule.Backend{
						Service: service.Service{
							Protocol: "http",
							Host:     "ip3.httpbin.zcorky.com",
							Port:     443,
						},
					},
				},
				{
					Path: "/ip2",
					Backend: rule.Backend{
						Service: service.Service{
							Protocol: "https",
							Host:     "ip2.httpbin.zcorky.com",
							Port:     443,
						},
					},
				},
			},
		},
	}

	s, err := MatchPath(rules[2].Paths, "/ip")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if s != nil {
		t.Fatalf("expected nil, got %v", s)
	}

	s, err = MatchPath(rules[2].Paths, "/ip1")
	if err != nil {
		t.Fatal(err)
	}
	if s.Host != "ip3.httpbin.zcorky.com" {
		t.Fatalf("expected ip3.httpbin.zcorky.com, got %s", s.Host)
	}
	if s.Port != 443 {
		t.Fatalf("expected 443, got %d", s.Port)
	}
	if s.Protocol != "http" {
		t.Fatalf("expected http, got %s", s.Protocol)
	}

	s, err = MatchPath(rules[2].Paths, "/ip2")
	if err != nil {
		t.Fatal(err)
	}
	if s.Host != "ip2.httpbin.zcorky.com" {
		t.Fatalf("expected ip2.httpbin.zcorky.com, got %s", s.Host)
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
					Host:     "task.$1.svc",
					Port:     8080,
				},
			},
		},
	}

	s, err := MatchHost(rules, "t-zero.example.work")
	if err != nil {
		t.Fatal(err)
	}
	if s.Service.Host != "task.zero.svc" {
		t.Fatalf("expected portainer, got %s", s.Service.Host)
	}
	if s.Service.Port != 8080 {
		t.Fatalf("expected 8080, got %d", s.Service.Port)
	}
	if s.Service.Protocol != "http" {
		t.Fatalf("expected http, got %s", s.Service.Protocol)
	}
}
