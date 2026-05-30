package core

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestRequestPrefersJSON(t *testing.T) {
	tests := []struct {
		accept string
		want   bool
	}{
		{"", false},
		{"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", false},
		{"application/json", true},
		{"application/json, */*", true},
		{"application/json;q=0.9", true},
		{"text/html, application/json;q=0.8", false},
		{"application/json, text/html;q=0.9", true},
		{"text/html, application/json;q=0.5", false},
	}

	for _, tt := range tests {
		req := &http.Request{Header: http.Header{"Accept": {tt.accept}}}
		if got := requestPrefersJSON(req); got != tt.want {
			t.Fatalf("Accept %q: got %v want %v", tt.accept, got, tt.want)
		}
	}
}

func TestIngressErrorPageJSON_SafeModeOmitsDetails(t *testing.T) {
	const secretHost = "internal-api.prod.example"
	raw := ingressErrorPageJSON(404, "Not Found", "The requested resource could not be found.", false, secretHost, "/admin", "GET", "resolver: no such host")

	var payload errorPageJSONBody
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Status != 404 || payload.Error != "Not Found" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if strings.Contains(raw, secretHost) || strings.Contains(raw, "resolver") {
		t.Fatal("safe mode must not echo sensitive fields")
	}
	if payload.Reason != "" || payload.Host != "" || payload.Path != "" || payload.Method != "" {
		t.Fatal("safe mode must omit verbose fields")
	}
}

func TestIngressErrorPageJSON_VerboseModeIncludesDetails(t *testing.T) {
	raw := ingressErrorPageJSON(404, "Not Found", "x", true, "host.example", "/p", "get", "err")
	var payload errorPageJSONBody
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Host != "host.example" || payload.Path != "/p" || payload.Method != "GET" || payload.Reason != "err" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestCompiledErrorPages_RenderWithNegotiation(t *testing.T) {
	pages, err := compileErrorPages(&Config{})
	if err != nil {
		t.Fatal(err)
	}

	html, ct := pages.RenderWithNegotiation(404, false, ErrorPageDetail{Hostname: "secret.example"}, false)
	if ct != errorPageContentTypeHTML || !strings.Contains(html, "<!DOCTYPE html>") {
		t.Fatalf("expected html page, got content-type %q", ct)
	}
	if strings.Contains(html, "secret.example") {
		t.Fatal("safe html must not echo host")
	}

	jsonBody, ct := pages.RenderWithNegotiation(404, false, ErrorPageDetail{Hostname: "secret.example"}, true)
	if ct != errorPageContentTypeJSON {
		t.Fatalf("expected json content type, got %q", ct)
	}
	if strings.Contains(jsonBody, "secret.example") {
		t.Fatal("safe json must not echo host")
	}
	if !strings.Contains(jsonBody, `"error":"Not Found"`) {
		t.Fatalf("expected structured json, got %q", jsonBody)
	}
}

func TestCompiledErrorPages_RenderWithNegotiation_InlineJSON(t *testing.T) {
	pages, err := compileErrorPages(&Config{
		ErrorPages: ErrorPages{
			Pages: map[string]ErrorPageSpec{
				"502": {Type: "inline", Body: `{"code":502,"msg":"upstream down"}`},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	body, ct := pages.RenderWithNegotiation(502, false, ErrorPageDetail{}, true)
	if ct != errorPageContentTypeJSON || body != `{"code":502,"msg":"upstream down"}` {
		t.Fatalf("got ct=%q body=%q", ct, body)
	}
}
