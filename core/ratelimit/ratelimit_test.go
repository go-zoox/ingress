package ratelimit

import (
	"net/http/httptest"
	"testing"

	"github.com/go-zoox/ingress/core/rule"
)

func TestCheck_IPPerRoute(t *testing.T) {
	p, err := compilePolicy(rule.RateLimit{
		Requests: 2,
		Period:   1,
		Key:      KeyIP,
	}, "test:rule", "", 0, "", "", 0, "")
	if err != nil {
		t.Fatalf("compilePolicy: %v", err)
	}

	req := httptest.NewRequest("GET", "http://example.com/", nil)
	req.RemoteAddr = "203.0.113.10:1234"

	for i := 0; i < 2; i++ {
		if blocked, _ := checkOne(req, p, 0, 0); blocked {
			t.Fatalf("request %d should pass", i+1)
		}
	}
	if blocked, retry := checkOne(req, p, 0, 0); !blocked {
		t.Fatal("third request should be blocked")
	} else if retry < 1 {
		t.Fatalf("expected retry-after >= 1, got %d", retry)
	}
}

func TestCheck_HeaderKeySeparateBuckets(t *testing.T) {
	p, err := compilePolicy(rule.RateLimit{
		Requests: 1,
		Period:   60,
		Key:      KeyHeader,
		Header:   "X-API-Key",
	}, "test:hdr", "", 0, "", "", 0, "")
	if err != nil {
		t.Fatalf("compilePolicy: %v", err)
	}

	reqA := httptest.NewRequest("GET", "http://example.com/", nil)
	reqA.Header.Set("X-API-Key", "alpha")
	if blocked, _ := checkOne(reqA, p, 0, 0); blocked {
		t.Fatal("first alpha request should pass")
	}
	if blocked, _ := checkOne(reqA, p, 0, 0); !blocked {
		t.Fatal("second alpha request should be blocked")
	}

	reqB := httptest.NewRequest("GET", "http://example.com/", nil)
	reqB.Header.Set("X-API-Key", "beta")
	if blocked, _ := checkOne(reqB, p, 0, 0); blocked {
		t.Fatal("beta bucket should be independent")
	}
}

func TestCompile_GlobalAndRule(t *testing.T) {
	enabled := true
	ing, err := Compile(
		rule.RateLimit{Enabled: &enabled, Requests: 100, Period: 60, Key: KeyGlobal},
		[]rule.Rule{{
			Host: "api.example.com",
			RateLimit: rule.RateLimit{
				Requests: 5,
				Period:   1,
				Key:      KeyIP,
			},
		}},
		"", 0, "", "", 0, "",
	)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if ing.Global == nil {
		t.Fatal("expected global limiter")
	}
	if ing.ByRule[0] == nil {
		t.Fatal("expected rule limiter")
	}
}

func TestCompile_InvalidKey(t *testing.T) {
	_, err := compilePolicy(rule.RateLimit{
		Requests: 1,
		Period:   1,
		Key:      "invalid",
	}, "test", "", 0, "", "", 0, "")
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
}

func TestCheck_GlobalAndRuleBothApply(t *testing.T) {
	global, err := compilePolicy(rule.RateLimit{
		Requests: 1,
		Period:   60,
		Key:      KeyGlobal,
	}, "test:global", "", 0, "", "", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	ruleP, err := compilePolicy(rule.RateLimit{
		Requests: 1,
		Period:   60,
		Key:      KeyIP,
	}, "test:rule", "", 0, "", "", 0, "")
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "http://example.com/", nil)
	req.RemoteAddr = "198.51.100.2:1234"

	if blocked, _ := Check(req, global, ruleP, 0); blocked {
		t.Fatal("first request should pass both limits")
	}
	if blocked, _ := Check(req, global, ruleP, 0); !blocked {
		t.Fatal("second request should hit global limit")
	}
}

func TestClientIP_TrustProxyXFF(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.5, 10.0.0.1")

	if got := ClientIP(req, true, 0); got != "203.0.113.5" {
		t.Fatalf("xff index 0: got %q", got)
	}
	if got := ClientIP(req, true, -1); got != "10.0.0.1" {
		t.Fatalf("xff index -1: got %q", got)
	}
	if got := ClientIP(req, false, 0); got != "10.0.0.1" {
		t.Fatalf("without trust_proxy: got %q", got)
	}
}

func TestCheck_RouteKeySeparateBuckets(t *testing.T) {
	p, err := compilePolicy(rule.RateLimit{
		Requests: 1,
		Period:   60,
		Key:      KeyRoute,
	}, "test:route", "", 0, "", "", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", "http://example.com/", nil)
	if blocked, _ := checkOne(req, p, 0, 0); blocked {
		t.Fatal("route 0 first should pass")
	}
	if blocked, _ := checkOne(req, p, 1, 1); blocked {
		t.Fatal("route 1 should have separate bucket")
	}
}
