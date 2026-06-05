package core

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/ingress/core/service"
)

func boolPtr(v bool) *bool { return &v }

func TestCompileRouterIndex_SkipsDisabledRules(t *testing.T) {
	disabled := false
	rules := []rule.Rule{
		{
			Host:    "blocked.example.com",
			Enabled: &disabled,
			Backend: rule.Backend{
				Service: service.Service{Name: "blocked", Port: 8080, Protocol: "http"},
			},
		},
		{
			Host: "live.example.com",
			Backend: rule.Backend{
				Service: service.Service{Name: "live", Port: 8080, Protocol: "http"},
			},
		},
	}

	idx, err := compileRouterIndex(rules, rule.Backend{})
	if err != nil {
		t.Fatal(err)
	}
	if len(idx.entries) != 1 {
		t.Fatalf("expected 1 compiled entry, got %d", len(idx.entries))
	}
	if idx.entries[0].ruleIndex != 1 {
		t.Fatalf("expected rule index 1, got %d", idx.entries[0].ruleIndex)
	}
}

func TestMatchHost_DisabledRuleSkipped(t *testing.T) {
	disabled := false
	rules := []rule.Rule{
		{
			Host:    "app.example.com",
			Enabled: &disabled,
			Backend: rule.Backend{
				Service: service.Service{Name: "old", Port: 8080, Protocol: "http"},
			},
		},
		{
			Host: "app.example.com",
			Backend: rule.Backend{
				Service: service.Service{Name: "new", Port: 8080, Protocol: "http"},
			},
		},
	}

	hm, err := MatchHost(rules, rule.Backend{}, "app.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if hm.Service == nil || hm.Service.Name != "new" {
		t.Fatalf("expected second enabled rule, got %+v", hm.Service)
	}
	if hm.ruleIndex != 1 {
		t.Fatalf("expected rule index 1, got %d", hm.ruleIndex)
	}
}

func TestBuild_DisabledRuleNotMatched(t *testing.T) {
	disabled := false
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host:    "app.example.com",
				Enabled: &disabled,
				Backend: rule.Backend{
					Redirect: rule.Redirect{URL: "https://old.example.com"},
				},
			},
			{
				Host: "app.example.com",
				Backend: rule.Backend{
					Handler: rule.Handler{Body: "ok"},
				},
			},
		},
	}

	c, err := New("test-version", cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ins := c.(*core)
	if err := ins.build(); err != nil {
		t.Fatalf("build: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://app.example.com/", nil)
	rec := httptest.NewRecorder()
	ins.app.ServeHTTP(rec, req)
	if rec.Code != 200 || rec.Body.String() != "ok" {
		t.Fatalf("expected handler from enabled rule, got %d body=%q", rec.Code, rec.Body.String())
	}
}

func TestListRouteRows_IncludesEnabledFlag(t *testing.T) {
	disabled := false
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host:    "off.example.com",
				Enabled: &disabled,
				Backend: rule.Backend{
					Service: service.Service{Name: "svc", Port: 8080, Protocol: "http"},
				},
			},
		},
	}
	rows, err := ListRouteRows(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].Enabled {
		t.Fatal("expected enabled=false on row")
	}
}

func TestRequestMatchesRoute_DisabledRuleFalse(t *testing.T) {
	disabled := false
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host:    "app.example.com",
				Enabled: &disabled,
				Backend: rule.Backend{
					Service: service.Service{Name: "svc", Port: 8080, Protocol: "http"},
				},
			},
		},
	}
	ok, err := RequestMatchesRoute(cfg, 0, -1, "app.example.com", "/")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("disabled rule should not match")
	}
}

func TestPreviewMatch_SkipsDisabledRule(t *testing.T) {
	disabled := false
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host:    "app.example.com",
				Enabled: &disabled,
				Backend: rule.Backend{
					Service: service.Service{Name: "old", Port: 8080, Protocol: "http"},
				},
			},
			{
				Host: "app.example.com",
				Backend: rule.Backend{
					Service: service.Service{Name: "new", Port: 8080, Protocol: "http"},
				},
			},
		},
	}
	prev, err := PreviewMatch(cfg, "app.example.com", "/")
	if err != nil {
		t.Fatal(err)
	}
	if !prev.Matched || prev.RuleIndex != 1 {
		t.Fatalf("expected match on rule 1, got %+v", prev)
	}
}
