package security

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-zoox/ingress/core/rule"
)

func TestCompileProfileStrict(t *testing.T) {
	ing, err := Compile(rule.Security{Profile: ProfileStrict}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !ing.Global.Active {
		t.Fatal("expected active global profile")
	}
	if ing.Global.Headers[headerXFrameOptions] != "DENY" {
		t.Fatalf("frame=%q", ing.Global.Headers[headerXFrameOptions])
	}
	if ing.Global.CORS != nil {
		t.Fatal("strict profile should not enable cors")
	}
}

func TestCompileProfileAPIRequiresOrigins(t *testing.T) {
	_, err := Compile(rule.Security{Profile: ProfileAPI}, nil)
	if err == nil {
		t.Fatal("expected cors origins error")
	}
	ing, err := Compile(rule.Security{
		Profile: ProfileAPI,
		CORS: rule.CORS{
			Origins: []string{"https://app.example.com"},
		},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if ing.Global.CORS == nil {
		t.Fatal("expected cors profile")
	}
}

func TestApplyHeadersHSTSOnlyOnHTTPS(t *testing.T) {
	ing, err := Compile(rule.Security{Profile: ProfileStrict}, nil)
	if err != nil {
		t.Fatal(err)
	}
	reqHTTP := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	hdr := http.Header{}
	ApplyHeaders(hdr, ing.Global, reqHTTP)
	if hdr.Get(headerStrictTransportSecurity) != "" {
		t.Fatal("hsts should not be set on http")
	}
}

func TestApplyHeadersHSTSOnForwardedHTTPS(t *testing.T) {
	ing, err := Compile(rule.Security{Profile: ProfileStrict}, nil)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	hdr := http.Header{}
	ApplyHeaders(hdr, ing.Global, req)
	if hdr.Get(headerStrictTransportSecurity) == "" {
		t.Fatal("expected hsts on forwarded https")
	}
}

func TestHandlePreflight(t *testing.T) {
	ing, err := Compile(rule.Security{
		Profile: ProfileAPI,
		CORS: rule.CORS{
			Origins: []string{"https://app.example.com"},
		},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodOptions, "https://api.example.com/v1", nil)
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()
	if !HandlePreflight(rec, req, ing.Global) {
		t.Fatal("expected preflight handled")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d", rec.Code)
	}
	if rec.Header().Get(headerAccessControlAllowOrigin) != "https://app.example.com" {
		t.Fatalf("acao=%q", rec.Header().Get(headerAccessControlAllowOrigin))
	}
}

func TestMergeSecurityPerPath(t *testing.T) {
	global := rule.Security{Profile: ProfileStrict}
	rules := []rule.Rule{{
		Security: rule.Security{Profile: ProfileStrict},
		Paths: []rule.Path{{
			Path: "/api",
			Security: rule.Security{
				Profile: ProfileEmbeddable,
			},
		}},
	}}
	ing, err := Compile(global, rules)
	if err != nil {
		t.Fatal(err)
	}
	if ing.ByRule[0].Headers[headerXFrameOptions] != "DENY" {
		t.Fatalf("rule frame=%q", ing.ByRule[0].Headers[headerXFrameOptions])
	}
	if ing.ByPath[0][0].Headers[headerXFrameOptions] != "SAMEORIGIN" {
		t.Fatalf("path frame=%q", ing.ByPath[0][0].Headers[headerXFrameOptions])
	}
}

func TestMergeSecurityPerRule(t *testing.T) {
	global := rule.Security{Profile: ProfileStrict}
	rules := []rule.Rule{{
		Security: rule.Security{
			Profile: ProfileEmbeddable,
		},
	}}
	ing, err := Compile(global, rules)
	if err != nil {
		t.Fatal(err)
	}
	if ing.ByRule[0].Headers[headerXFrameOptions] != "SAMEORIGIN" {
		t.Fatalf("got %q", ing.ByRule[0].Headers[headerXFrameOptions])
	}
}
