package service

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-zoox/zoox"
)

// TestRequestScheme verifies scheme detection for auto-generated OAuth2 redirect URLs,
// including the X-Forwarded-Proto case behind a TLS-terminating reverse proxy.
func TestRequestScheme(t *testing.T) {
	cases := []struct {
		name string
		tls  bool
		xfp  string
		want string
	}{
		{name: "direct tls", tls: true, xfp: "", want: "https"},
		{name: "forwarded https", tls: false, xfp: "https", want: "https"},
		{name: "forwarded http", tls: false, xfp: "http", want: "http"},
		{name: "no headers", tls: false, xfp: "", want: "http"},
		{name: "forwarded case-insensitive", tls: false, xfp: "HTTPS", want: "https"},
		{name: "forwarded trimmed", tls: false, xfp: "  https  ", want: "https"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
			if tc.tls {
				req.TLS = &tls.ConnectionState{}
			}
			if tc.xfp != "" {
				req.Header.Set("X-Forwarded-Proto", tc.xfp)
			}
			if got := requestScheme(req); got != tc.want {
				t.Fatalf("requestScheme(): got %q, want %q", got, tc.want)
			}
		})
	}
}

// runOAuth2Flow builds a one-shot zoox app that runs fn against a fresh Context.
func runOAuth2Flow(t *testing.T, fn func(ctx *zoox.Context)) {
	t.Helper()
	app := zoox.New()
	app.Use(func(ctx *zoox.Context) {
		fn(ctx)
		ctx.String(http.StatusOK, "done")
	})
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

// TestOAuth2FlowMap_ConcurrentFlows verifies that multiple in-flight logins on the same
// session each keep their own state -> redirect mapping (no single-slot clobber).
func TestOAuth2FlowMap_ConcurrentFlows(t *testing.T) {
	var r1, r2 string
	var ok1, ok2, okMissing, okAgain bool

	runOAuth2Flow(t, func(ctx *zoox.Context) {
		saveOAuth2Flow(ctx, "state1", "/app/one")
		saveOAuth2Flow(ctx, "state2", "/app/two")

		r1, ok1 = takeOAuth2Flow(ctx, "state1")
		r2, ok2 = takeOAuth2Flow(ctx, "state2")
		_, okMissing = takeOAuth2Flow(ctx, "missing-state")
		_, okAgain = takeOAuth2Flow(ctx, "state1") // already consumed
	})

	if !ok1 || r1 != "/app/one" {
		t.Fatalf("state1: ok=%v redirect=%q, want ok=true redirect='/app/one'", ok1, r1)
	}
	if !ok2 || r2 != "/app/two" {
		t.Fatalf("state2: ok=%v redirect=%q, want ok=true redirect='/app/two'", ok2, r2)
	}
	if okMissing {
		t.Fatal("unknown state should not be found")
	}
	if okAgain {
		t.Fatal("consumed state must not be found again")
	}
}

// TestOAuth2FlowMap_LegacyFallback verifies in-flight flows created by older versions
// (single state/redirect slot) still resolve across a rolling upgrade.
func TestOAuth2FlowMap_LegacyFallback(t *testing.T) {
	var r string
	var ok, again bool

	runOAuth2Flow(t, func(ctx *zoox.Context) {
		ctx.Session().Set(sessOAuth2State, "legacy-state")
		ctx.Session().Set(sessOAuth2Redirect, "/legacy/path")

		r, ok = takeOAuth2Flow(ctx, "legacy-state")
		_, again = takeOAuth2Flow(ctx, "legacy-state")
	})

	if !ok || r != "/legacy/path" {
		t.Fatalf("legacy: ok=%v redirect=%q, want ok=true redirect='/legacy/path'", ok, r)
	}
	if again {
		t.Fatal("legacy state should be consumed after take")
	}
}

// TestOAuth2FlowMap_Bounded verifies writeOAuth2Flows keeps the session cookie bounded
// when more flows than maxOAuth2Flows are initiated.
func TestOAuth2FlowMap_Bounded(t *testing.T) {
	app := zoox.New()
	var count int
	app.Use(func(ctx *zoox.Context) {
		for i := 0; i < maxOAuth2Flows+20; i++ {
			saveOAuth2Flow(ctx, stateForIndex(i), "/p")
		}
		count = len(readOAuth2Flows(ctx))
		ctx.String(http.StatusOK, "done")
	})
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if count > maxOAuth2Flows {
		t.Fatalf("expected at most %d flows, got %d", maxOAuth2Flows, count)
	}
	if count == 0 {
		t.Fatal("expected some flows to remain")
	}
}

func stateForIndex(i int) string {
	hex := "0123456789abcdef"
	buf := make([]byte, 32)
	for j := range buf {
		buf[j] = hex[(i+j)%len(hex)]
	}
	return string(buf)
}
