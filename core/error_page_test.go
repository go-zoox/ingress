package core

import (
	"strings"
	"testing"
)

func TestIngressErrorPageHTML_SafeModeOmitsDetails(t *testing.T) {
	const secretHost = "internal-api.prod.example"
	html := ingressErrorPageHTML(404, "Not Found", "The requested resource could not be found.", false, secretHost, "/admin", "GET", "resolver: no such host")
	if strings.Contains(html, secretHost) {
		t.Fatal("safe mode must not echo Host")
	}
	if strings.Contains(html, "resolver") {
		t.Fatal("safe mode must not echo error reason")
	}
	if strings.Contains(html, "<dt>Host</dt>") {
		t.Fatal("safe mode must not render request detail list")
	}
}

func TestIngressErrorPageHTML_VerboseModeIncludesEscapedHost(t *testing.T) {
	html := ingressErrorPageHTML(404, "Route not found", "x", true, `a<b`, "/p", "GET", "err")
	if !strings.Contains(html, "a&lt;b") {
		t.Fatal("expected escaped host")
	}
	if !strings.Contains(html, "<dt>Host</dt>") {
		t.Fatal("verbose mode should list host")
	}
}
