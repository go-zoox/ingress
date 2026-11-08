package core

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/ingress/core/waf"
)

// newBanTestCore builds a core with a single handler route so requests can be
// served end-to-end without an upstream.
func newBanTestCore(t *testing.T, cfg *Config) *core {
	t.Helper()
	c, err := New("test-version", cfg)
	if err != nil {
		t.Fatalf("failed to create core: %v", err)
	}
	ins, ok := c.(*core)
	if !ok {
		t.Fatal("failed to cast core instance")
	}
	if err := ins.build(); err != nil {
		t.Fatalf("failed to build core: %v", err)
	}
	return ins
}

func handlerConfig(host string) *Config {
	return &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: host,
				Backend: rule.Backend{
					Type:    backendTypeHandler,
					Handler: rule.Handler{StatusCode: 200, Body: "ok"},
				},
			},
		},
	}
}

func serveBanRequest(ins *core, host, remoteAddr, userAgent string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "http://"+host+"/", nil)
	req.RemoteAddr = remoteAddr
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	rec := httptest.NewRecorder()
	ins.app.ServeHTTP(rec, req)
	return rec
}

func mustBanSet(t *testing.T, entries []waf.BanEntry) *waf.BanSet {
	t.Helper()
	set, err := waf.CompileBans(entries, nil)
	if err != nil {
		t.Fatalf("CompileBans: %v", err)
	}
	return set
}

func TestBuild_BanBlocksIP(t *testing.T) {
	ins := newBanTestCore(t, handlerConfig("api.example.com"))

	// Without a ban set the same request succeeds.
	if rec := serveBanRequest(ins, "api.example.com", "203.0.113.42:1234", ""); rec.Code != 200 {
		t.Fatalf("pre-ban: expected 200, got %d", rec.Code)
	}

	ins.SetBanSet(mustBanSet(t, []waf.BanEntry{
		{ID: 1, Kind: waf.BanKindIP, Value: "203.0.113.42"},
	}))

	rec := serveBanRequest(ins, "api.example.com", "203.0.113.42:1234", "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("banned IP: expected 403, got %d", rec.Code)
	}

	// Another IP is unaffected.
	if rec := serveBanRequest(ins, "api.example.com", "203.0.113.99:1234", ""); rec.Code != 200 {
		t.Fatalf("other IP: expected 200, got %d", rec.Code)
	}
}

func TestBuild_BanBlockedRequestLogsBanBlock(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://api.example.com/", nil)
	req.RemoteAddr = "203.0.113.42:1234"
	req.Header.Set("User-Agent", "sqlmap/1.7.2")

	extra := buildAccessLogExtraFields(req, accessLogMeta{
		BanBlock:       true,
		UpstreamStatus: http.StatusForbidden,
	})
	if !strings.Contains(extra, "ban_block=1") {
		t.Fatalf("expected ban_block=1 in %q", extra)
	}
	// A ban hit must not be reported as a config-level WAF block.
	if !strings.Contains(extra, "waf_block=0") {
		t.Fatalf("ban hit must not set waf_block, got %q", extra)
	}

	// And the default (no ban, no WAF) shape keeps ban_block=0.
	plain := buildAccessLogExtraFields(req, accessLogMeta{UpstreamStatus: 200})
	if !strings.Contains(plain, "ban_block=0") {
		t.Fatalf("expected ban_block=0 in %q", plain)
	}
}

func TestBuild_BanSurvivesWAFDisable(t *testing.T) {
	ins := newBanTestCore(t, handlerConfig("api.example.com"))
	ins.SetBanSet(mustBanSet(t, []waf.BanEntry{
		{ID: 1, Kind: waf.BanKindIP, Value: "203.0.113.42"},
	}))

	// Emergency "turn WAF off" must not disable the ban list.
	off := false
	ins.SetWAFOverride(&off)

	rec := serveBanRequest(ins, "api.example.com", "203.0.113.42:1234", "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("ban must survive WAF override=false: expected 403, got %d", rec.Code)
	}
}

func TestBuild_BanOverridesAllowList(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "admin.example.com",
				WAFPatch: map[string]any{
					"enabled":         true,
					"disable_builtin": true,
					"allow":           []any{"203.0.113.0/24"},
				},
				Backend: rule.Backend{
					Type:    backendTypeHandler,
					Handler: rule.Handler{StatusCode: 200, Body: "ok"},
				},
			},
		},
	}
	ins := newBanTestCore(t, cfg)

	// Allowlisted source passes before the ban exists.
	if rec := serveBanRequest(ins, "admin.example.com", "203.0.113.42:1234", ""); rec.Code != 200 {
		t.Fatalf("allowlisted before ban: expected 200, got %d", rec.Code)
	}

	ins.SetBanSet(mustBanSet(t, []waf.BanEntry{
		{ID: 1, Kind: waf.BanKindIP, Value: "203.0.113.42"},
	}))

	rec := serveBanRequest(ins, "admin.example.com", "203.0.113.42:1234", "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("ban must override allow list: expected 403, got %d", rec.Code)
	}
}

func TestBuild_BanHostScope(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "api.example.com",
				Backend: rule.Backend{
					Type:    backendTypeHandler,
					Handler: rule.Handler{StatusCode: 200, Body: "api"},
				},
			},
			{
				Host: "www.example.com",
				Backend: rule.Backend{
					Type:    backendTypeHandler,
					Handler: rule.Handler{StatusCode: 200, Body: "www"},
				},
			},
		},
	}
	ins := newBanTestCore(t, cfg)
	ins.SetBanSet(mustBanSet(t, []waf.BanEntry{
		{ID: 1, Kind: waf.BanKindIP, Value: "203.0.113.42", Hosts: []string{"api.example.com"}},
	}))

	if rec := serveBanRequest(ins, "api.example.com", "203.0.113.42:1234", ""); rec.Code != http.StatusForbidden {
		t.Fatalf("scoped host: expected 403, got %d", rec.Code)
	}
	if rec := serveBanRequest(ins, "www.example.com", "203.0.113.42:1234", ""); rec.Code != 200 {
		t.Fatalf("other host must be unaffected: expected 200, got %d", rec.Code)
	}
}

func TestBuild_BanUAContains(t *testing.T) {
	ins := newBanTestCore(t, handlerConfig("api.example.com"))
	ins.SetBanSet(mustBanSet(t, []waf.BanEntry{
		{ID: 1, Kind: waf.BanKindUAContains, Value: "sqlmap"},
	}))

	rec := serveBanRequest(ins, "api.example.com", "198.51.100.9:1234", "sqlmap/1.7.2#stable")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("banned UA: expected 403, got %d", rec.Code)
	}
	if rec := serveBanRequest(ins, "api.example.com", "198.51.100.9:1234", "curl/8.4.0"); rec.Code != 200 {
		t.Fatalf("normal UA: expected 200, got %d", rec.Code)
	}
}

func TestBuild_BanTTLExpires(t *testing.T) {
	ins := newBanTestCore(t, handlerConfig("api.example.com"))
	ins.SetBanSet(mustBanSet(t, []waf.BanEntry{
		{ID: 1, Kind: waf.BanKindIP, Value: "203.0.113.42", ExpiresAt: time.Now().Add(-time.Second)},
	}))

	if rec := serveBanRequest(ins, "api.example.com", "203.0.113.42:1234", ""); rec.Code != 200 {
		t.Fatalf("expired ban must not block: expected 200, got %d", rec.Code)
	}
}

func TestBuild_BanSetNilResets(t *testing.T) {
	ins := newBanTestCore(t, handlerConfig("api.example.com"))
	ins.SetBanSet(mustBanSet(t, []waf.BanEntry{
		{ID: 1, Kind: waf.BanKindIP, Value: "203.0.113.42"},
	}))
	if rec := serveBanRequest(ins, "api.example.com", "203.0.113.42:1234", ""); rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 while banned, got %d", rec.Code)
	}

	// Clearing the snapshot restores access with no reload involved.
	ins.SetBanSet(nil)
	if rec := serveBanRequest(ins, "api.example.com", "203.0.113.42:1234", ""); rec.Code != 200 {
		t.Fatalf("after clearing ban set: expected 200, got %d", rec.Code)
	}
}

func TestBuild_BanPrecedesRateLimit(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		RateLimit: rule.RateLimit{
			Requests: 1,
			Period:   60,
			Key:      "ip",
		},
		Rules: []rule.Rule{
			{
				Host: "api.example.com",
				Backend: rule.Backend{
					Type:    backendTypeHandler,
					Handler: rule.Handler{StatusCode: 200, Body: "ok"},
				},
			},
		},
	}
	ins := newBanTestCore(t, cfg)
	ins.SetBanSet(mustBanSet(t, []waf.BanEntry{
		{ID: 1, Kind: waf.BanKindIP, Value: "203.0.113.42"},
	}))

	// Exhaust the rate limit with a different IP first, so the ban check has a
	// real chance to be shadowed if it ran after rate limiting.
	for i := 0; i < 3; i++ {
		serveBanRequest(ins, "api.example.com", "198.51.100.7:1234", "")
	}

	rec := serveBanRequest(ins, "api.example.com", "203.0.113.42:1234", "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("banned IP must get 403 not 429, got %d", rec.Code)
	}
}

func TestBuild_BanSetSwapIsConcurrencySafe(t *testing.T) {
	// The snapshot is swapped atomically while requests read it. Run with -race:
	// this guards against regressing to a non-atomic field (the existing
	// SetWAFCallback writes a plain field and is not concurrency-safe).
	ins := newBanTestCore(t, handlerConfig("api.example.com"))

	ban := mustBanSet(t, []waf.BanEntry{{ID: 1, Kind: waf.BanKindIP, Value: "203.0.113.42"}})
	clear := mustBanSet(t, []waf.BanEntry{{ID: 2, Kind: waf.BanKindIP, Value: "203.0.113.99"}})

	stop := make(chan struct{})
	var wg sync.WaitGroup

	// Writer: keep swapping snapshots.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				ins.SetBanSet(ban)
				ins.SetBanSet(clear)
			}
		}
	}()

	// Readers: hit the request path while swaps happen.
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				rec := serveBanRequest(ins, "api.example.com", "198.51.100.7:1234", "")
				// Either snapshot may be active; both are valid outcomes.
				if rec.Code != 200 && rec.Code != http.StatusForbidden {
					t.Errorf("unexpected status %d", rec.Code)
					return
				}
			}
		}()
	}

	time.Sleep(20 * time.Millisecond)
	close(stop)
	wg.Wait()
}
