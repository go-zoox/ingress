package core

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-zoox/ingress/core/peerip"
	"github.com/go-zoox/zoox"
)

func TestPeerIPMiddleware_CapturesTCPPeerBeforeRealIPRewrite(t *testing.T) {
	app := newIngressApplication()
	app.Use(func(ctx *zoox.Context) {
		ctx.String(http.StatusOK, peerip.FromRequest(ctx.Request)+"|"+ctx.Request.RemoteAddr)
	})

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	req.RemoteAddr = "198.51.100.23:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.7")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Captured peer is the TCP source; RemoteAddr keeps zoox realip semantics.
	if got := rec.Body.String(); got != "198.51.100.23|203.0.113.7" {
		t.Fatalf("expected captured peer and rewritten RemoteAddr, got %q", got)
	}
}
