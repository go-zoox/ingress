package core

import (
	"fmt"
	"testing"

	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/ingress/core/service"
)

func BenchmarkMatchHostIndex_lastExactRule(b *testing.B) {
	rules := make([]rule.Rule, 100)
	for i := range rules {
		rules[i] = rule.Rule{
			Host: fmt.Sprintf("host-%d.example.com", i),
			Backend: rule.Backend{
				Service: service.Service{
					Protocol: "http",
					Name:     "svc",
					Port:     8080,
				},
			},
		}
	}
	idx, err := compileRouterIndex(rules, rule.Backend{})
	if err != nil {
		b.Fatal(err)
	}
	host := "host-99.example.com"
	fallback := rule.Backend{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := matchHostIndex(idx, rules, fallback, host); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMatchPathWithRouter_lastPath(b *testing.B) {
	paths := make([]rule.Path, 50)
	for i := range paths {
		paths[i] = rule.Path{
			Path: fmt.Sprintf("/p%d", i),
			Backend: rule.Backend{
				Service: service.Service{
					Protocol: "http",
					Name:     "svc",
					Port:     8080,
				},
			},
		}
	}
	rules := []rule.Rule{
		{
			Host: "a.example.com",
			Backend: rule.Backend{
				Service: service.Service{
					Protocol: "http",
					Name:     "default",
					Port:     8080,
				},
			},
			Paths: paths,
		},
	}
	idx, err := compileRouterIndex(rules, rule.Backend{})
	if err != nil {
		b.Fatal(err)
	}
	const wantPath = "/p49"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := matchPathWithRouter(idx, rules, 0, wantPath); err != nil {
			b.Fatal(err)
		}
	}
}
