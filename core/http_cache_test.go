package core

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-zoox/cache"
	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/zoox"
)

func TestCanonicalQuerySorted(t *testing.T) {
	q := canonicalQuerySorted("b=2&a=1&b=1")
	if q != "a=1&b=1&b=2" {
		t.Fatalf("got %q", q)
	}
	if canonicalQuerySorted("") != "" {
		t.Fatal("empty")
	}
}

func TestBuildHTTPCacheStorageKeyStableForHEADvsGET(t *testing.T) {
	bc := rule.BackendCache{Enabled: true, TTL: 60, KeyHeaders: []string{}}
	pc := normalizeHTTPCache(bc)
	if pc == nil {
		t.Fatal("nil pc")
	}
	pc.KeyHeaders = nil

	reqG := httptest.NewRequest(http.MethodGet, "http://example.com/path?z=1&a=2", nil)
	reqH := httptest.NewRequest(http.MethodHead, "http://example.com/path?z=1&a=2", nil)
	k1 := buildHTTPCacheStorageKey(reqG, "example.com", "/path", pc)
	k2 := buildHTTPCacheStorageKey(reqH, "example.com", "/path", pc)
	if k1 != k2 {
		t.Fatalf("HEAD/GET keys differ: %s vs %s", k1, k2)
	}
}

func TestNormalizeHTTPCacheDisabled(t *testing.T) {
	if normalizeHTTPCache(rule.BackendCache{Enabled: false}) != nil {
		t.Fatal("expected nil")
	}
}

func TestHTTPCacheRequestBypassesNoCache(t *testing.T) {
	bc := rule.BackendCache{Enabled: true}
	pc := normalizeHTTPCache(bc)
	if pc == nil {
		t.Fatal("pc")
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Cache-Control", "no-cache")
	if !httpCacheRequestBypasses(req, pc) {
		t.Fatal("expected bypass")
	}
}

func TestHTTPCacheRequestBypassesPragma(t *testing.T) {
	bc := rule.BackendCache{Enabled: true}
	pc := normalizeHTTPCache(bc)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Pragma", "no-cache")
	if !httpCacheRequestBypasses(req, pc) {
		t.Fatal("expected pragma bypass")
	}
}

func TestHTTPCacheRequestBypassesRange(t *testing.T) {
	bc := rule.BackendCache{Enabled: true}
	pc := normalizeHTTPCache(bc)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Range", "bytes=0-1")
	if !httpCacheRequestBypasses(req, pc) {
		t.Fatal("expected range bypass")
	}
}

// --- Integration: real go-zoox/cache memory backend + Zoox ctx.Cache() ---

func TestHTTPCache_MemoryBackendSetGet(t *testing.T) {
	c := cache.New()
	key := httpCacheKeyPrefix + "mem-roundtrip"
	want := &httpCacheEntry{
		StatusCode: http.StatusOK,
		Header:     map[string][]string{"Content-Type": {"text/plain"}, "X-Echo": {"a", "b"}},
		Body:       []byte(`{"hello":"world"}`),
	}
	if err := c.Set(key, want, time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	var got httpCacheEntry
	if err := c.Get(key, &got); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.StatusCode != want.StatusCode {
		t.Fatalf("status got %d want %d", got.StatusCode, want.StatusCode)
	}
	if string(got.Body) != string(want.Body) {
		t.Fatalf("body got %q want %q", got.Body, want.Body)
	}
	if len(got.Header["Content-Type"]) != 1 || got.Header["Content-Type"][0] != "text/plain" {
		t.Fatalf("Content-Type: %+v", got.Header["Content-Type"])
	}
	if len(got.Header["X-Echo"]) != 2 {
		t.Fatalf("X-Echo: %+v", got.Header["X-Echo"])
	}
}

// Redis path JSON-marshals the entry; ensure the stored shape round-trips like production Redis decode.
func TestHTTPCache_EntryJSONRoundTrip(t *testing.T) {
	ent := &httpCacheEntry{
		StatusCode: http.StatusTemporaryRedirect,
		Header:     map[string][]string{"Location": {"https://upstream.example/where"}},
		Body:       nil,
	}
	raw, err := json.Marshal(ent)
	if err != nil {
		t.Fatal(err)
	}
	var got httpCacheEntry
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("status %d", got.StatusCode)
	}
	if got.Header["Location"][0] != "https://upstream.example/where" {
		t.Fatalf("Location %+v", got.Header["Location"])
	}
}

func TestHTTPCache_TryServeHit_ZooxMemoryCache(t *testing.T) {
	app := zoox.New()
	cacheKey := httpCacheKeyPrefix + "zoox-hit"
	stored := &httpCacheEntry{
		StatusCode: http.StatusOK,
		Header: map[string][]string{
			"Content-Type": {"application/json"},
			"X-Cache-Test": {"1"},
		},
		Body: []byte(`{"cached":true}`),
	}
	if err := app.Cache().Set(cacheKey, stored, 5*time.Minute); err != nil {
		t.Fatalf("seed cache: %v", err)
	}

	var served bool
	app.Use(func(ctx *zoox.Context) {
		pc := normalizeHTTPCache(rule.BackendCache{Enabled: true, TTL: 300})
		ok, code, n := tryServeHTTPCache(ctx, pc, cacheKey)
		if !ok || code != http.StatusOK || n != int64(len(stored.Body)) {
			t.Errorf("tryServeHTTPCache ok=%v code=%d n=%d want ok true code=200 n=%d", ok, code, n, len(stored.Body))
		}
		served = ok
		if ok {
			return
		}
		ctx.String(http.StatusTeapot, "miss")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/api", nil)
	app.ServeHTTP(rec, req)

	if !served {
		t.Fatal("expected cache hit path")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("response code %d body %q", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != `{"cached":true}` {
		t.Fatalf("body %q", rec.Body.String())
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("Content-Type %q", rec.Header().Get("Content-Type"))
	}
	if rec.Header().Get("X-Cache-Test") != "1" {
		t.Fatalf("X-Cache-Test %q", rec.Header().Get("X-Cache-Test"))
	}
}

func TestHTTPCache_TryServeMiss_ZooxMemoryCache(t *testing.T) {
	app := zoox.New()
	cacheKey := httpCacheKeyPrefix + "zoox-missing"
	var miss bool
	app.Use(func(ctx *zoox.Context) {
		pc := normalizeHTTPCache(rule.BackendCache{Enabled: true, TTL: 300})
		ok, _, _ := tryServeHTTPCache(ctx, pc, cacheKey)
		miss = !ok
		if !ok {
			ctx.String(http.StatusNoContent, "")
		}
	})
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if !miss {
		t.Fatal("expected miss")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestHTTPCache_TryServeBypassWithCachedEntry_Zoox(t *testing.T) {
	app := zoox.New()
	cacheKey := httpCacheKeyPrefix + "zoox-bypass"
	if err := app.Cache().Set(cacheKey, &httpCacheEntry{
		StatusCode: http.StatusOK,
		Header:     map[string][]string{"Content-Type": {"text/plain"}},
		Body:       []byte("should-not-serve"),
	}, time.Minute); err != nil {
		t.Fatal(err)
	}

	var ranFallback bool
	app.Use(func(ctx *zoox.Context) {
		pc := normalizeHTTPCache(rule.BackendCache{Enabled: true, TTL: 300})
		ctx.Request.Header.Set("Cache-Control", "no-cache")
		if hit, _, _ := tryServeHTTPCache(ctx, pc, cacheKey); hit {
			t.Error("unexpected cache hit when client sent no-cache")
			return
		}
		ranFallback = true
		ctx.String(http.StatusTeapot, "bypass")
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Cache-Control", "no-cache")
	app.ServeHTTP(rec, req)
	if !ranFallback {
		t.Fatal("expected fallback")
	}
	if rec.Code != http.StatusTeapot || rec.Body.String() != "bypass" {
		t.Fatalf("got %d %q", rec.Code, rec.Body.String())
	}
}

func TestHTTPCache_TryServeRedirectHit_Zoox(t *testing.T) {
	app := zoox.New()
	cacheKey := httpCacheKeyPrefix + "zoox-redirect"
	if err := app.Cache().Set(cacheKey, &httpCacheEntry{
		StatusCode: http.StatusFound,
		Header:     map[string][]string{"Location": {"https://example.com/after"}},
		Body:       nil,
	}, time.Minute); err != nil {
		t.Fatal(err)
	}

	app.Use(func(ctx *zoox.Context) {
		pc := normalizeHTTPCache(rule.BackendCache{Enabled: true, TTL: 300})
		if hit, _, _ := tryServeHTTPCache(ctx, pc, cacheKey); hit {
			return
		}
		ctx.String(http.StatusInternalServerError, "no hit")
	})
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusFound {
		t.Fatalf("status %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "https://example.com/after" {
		t.Fatalf("Location %q", loc)
	}
}

func TestHTTPCache_TryServeHEADFromGETEntry_Zoox(t *testing.T) {
	app := zoox.New()
	cacheKey := httpCacheKeyPrefix + "zoox-head"
	body := []byte(`full`)
	if err := app.Cache().Set(cacheKey, &httpCacheEntry{
		StatusCode: http.StatusOK,
		Header:     map[string][]string{"Content-Type": {"text/plain"}},
		Body:       body,
	}, time.Minute); err != nil {
		t.Fatal(err)
	}

	app.Use(func(ctx *zoox.Context) {
		pc := normalizeHTTPCache(rule.BackendCache{Enabled: true, TTL: 300})
		if hit, _, _ := tryServeHTTPCache(ctx, pc, cacheKey); hit {
			return
		}
		ctx.String(http.StatusInternalServerError, "no hit")
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodHead, "http://127.0.0.1/doc", nil)
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("HEAD body should be empty, got %q", rec.Body.String())
	}
	if rec.Header().Get("Content-Length") != "4" {
		t.Fatalf("Content-Length %q", rec.Header().Get("Content-Length"))
	}
}
