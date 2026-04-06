package core

import (
	"testing"

	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/ingress/core/service"
)

func TestEffectiveHostType(t *testing.T) {
	tests := []struct {
		declared string
		host     string
		want     string
	}{
		{"", "a.b.com", "exact"},
		{"", `(\w+).inlets.example.com`, "regex"},
		{"", "*.inlets.example.com", "wildcard"},
		{"", "^.*\\.example\\.com$", "regex"},
		{"exact", `(\w+).x.com`, "exact"},
		{"regex", "plain.host", "regex"},
		{"wildcard", "*.x.com", "wildcard"},
		{"auto", "only.dots", "exact"},
		{"auto", "*.x.com", "wildcard"},
	}
	for _, tt := range tests {
		if got := effectiveHostType(tt.declared, tt.host); got != tt.want {
			t.Errorf("effectiveHostType(%q, %q) = %q, want %q", tt.declared, tt.host, got, tt.want)
		}
	}
}

func TestCompileRouterIndexInfersHostType(t *testing.T) {
	rules := []rule.Rule{
		{
			Host: `(\w+).inlets.example.com`,
			Backend: rule.Backend{
				Service: service.Service{Name: "inlets", Port: 8080},
			},
		},
	}
	_, err := compileRouterIndex(rules, rule.Backend{})
	if err != nil {
		t.Fatal(err)
	}
	if rules[0].HostType != "regex" {
		t.Fatalf("HostType = %q, want regex", rules[0].HostType)
	}
}
