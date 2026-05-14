package waf

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-zoox/ingress/core/rule"
)

func TestCheckRequest_IPDeny_Block(t *testing.T) {
	t.Parallel()
	m := rule.WAF{Enabled: true, DisableBuiltin: true, Deny: []string{"192.168.3.33"}}
	prof, err := compileProfile(0, m)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://x/", nil)
	req.RemoteAddr = "192.168.3.33:4444"
	if !CheckRequest(prof, req, "x", "/", http.MethodGet) {
		t.Fatal("expected block")
	}
}

func TestCheckRequest_CustomSignature(t *testing.T) {
	t.Parallel()
	m := rule.WAF{
		Enabled:        true,
		DisableBuiltin: true,
		Rules: []rule.WAFRule{{
			ID:      "t1",
			Type:    PatternTypeContains,
			Pattern: "BADTOKEN",
			Targets: []string{TargetQuery},
		}},
	}
	prof, err := compileProfile(0, m)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://x/p?x=BADTOKEN", nil)
	req.RemoteAddr = "127.0.0.1:1"
	if !CheckRequest(prof, req, "x", "/p", http.MethodGet) {
		t.Fatal("expected block on query token")
	}
}

func TestCheckRequest_LogOnly_NoBlock(t *testing.T) {
	t.Parallel()
	m := rule.WAF{
		Enabled:        true,
		LogOnly:        true,
		DisableBuiltin: true,
		Rules: []rule.WAFRule{{
			ID:      "t1",
			Type:    PatternTypeContains,
			Pattern: "BADTOKEN",
			Targets: []string{TargetQuery},
		}},
	}
	prof, err := compileProfile(0, m)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://x/p?x=BADTOKEN", nil)
	req.RemoteAddr = "127.0.0.1:1"
	if CheckRequest(prof, req, "x", "/p", http.MethodGet) {
		t.Fatal("log_only should not block")
	}
}

func TestCheckRequest_TrustProxy_XFF(t *testing.T) {
	t.Parallel()
	m := rule.WAF{
		Enabled:        true,
		TrustProxy:     true,
		XFFIndex:       0,
		DisableBuiltin: true,
		Deny:           []string{"203.0.113.55"},
	}
	prof, err := compileProfile(0, m)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://x/", nil)
	req.RemoteAddr = "127.0.0.1:1"
	req.Header.Set(headerXForwardedFor, "203.0.113.55, 10.0.0.1")
	if !CheckRequest(prof, req, "x", "/", http.MethodGet) {
		t.Fatal("expected deny via XFF leftmost")
	}
}

func TestCheckRequest_AllowWhitelistBlocksOther(t *testing.T) {
	t.Parallel()
	m := rule.WAF{
		Enabled:        true,
		DisableBuiltin: true,
		Allow:          []string{"10.10.10.10"},
	}
	prof, err := compileProfile(0, m)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://x/", nil)
	req.RemoteAddr = "192.168.1.1:1"
	if !CheckRequest(prof, req, "x", "/", http.MethodGet) {
		t.Fatal("expected block: not on allowlist")
	}
	req2 := httptest.NewRequest(http.MethodGet, "http://x/", nil)
	req2.RemoteAddr = "10.10.10.10:1111"
	if CheckRequest(prof, req2, "x", "/", http.MethodGet) {
		t.Fatal("allowlisted IP must pass ip phase")
	}
}

func TestCheckRequest_NilOrDisabled_NoBlock(t *testing.T) {
	t.Parallel()
	if CheckRequest(nil, httptest.NewRequest(http.MethodGet, "http://x/", nil), "x", "/", http.MethodGet) {
		t.Fatal("nil profile must not block")
	}
	disabled, err := compileProfile(0, rule.WAF{Enabled: false})
	if err != nil {
		t.Fatal(err)
	}
	if CheckRequest(disabled, httptest.NewRequest(http.MethodGet, "http://x/", nil), "x", "/", http.MethodGet) {
		t.Fatal("disabled profile must not block")
	}
	en, err := compileProfile(0, rule.WAF{Enabled: true, DisableBuiltin: true})
	if err != nil {
		t.Fatal(err)
	}
	if CheckRequest(en, nil, "x", "/", http.MethodGet) {
		t.Fatal("nil request must not block")
	}
}

func TestCheckRequest_IPDeny_CIDR(t *testing.T) {
	t.Parallel()
	m := rule.WAF{
		Enabled:        true,
		DisableBuiltin: true,
		Deny:           []string{"198.51.100.0/24"},
	}
	prof, err := compileProfile(0, m)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://x/", nil)
	req.RemoteAddr = "198.51.100.88:1234"
	if !CheckRequest(prof, req, "x", "/", http.MethodGet) {
		t.Fatal("expected block inside CIDR")
	}
	req2 := httptest.NewRequest(http.MethodGet, "http://x/", nil)
	req2.RemoteAddr = "203.0.113.1:1"
	if CheckRequest(prof, req2, "x", "/", http.MethodGet) {
		t.Fatal("expected pass outside CIDR")
	}
}

func TestCheckRequest_GlobalLogOnly_IPDeny_NoBlock(t *testing.T) {
	t.Parallel()
	m := rule.WAF{
		Enabled:        true,
		LogOnly:        true,
		DisableBuiltin: true,
		Deny:           []string{"192.0.2.9"},
	}
	prof, err := compileProfile(0, m)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://x/", nil)
	req.RemoteAddr = "192.0.2.9:1"
	if CheckRequest(prof, req, "x", "/", http.MethodGet) {
		t.Fatal("global log_only must not block ip deny")
	}
}

func TestCheckRequest_XFFIndex_NegativeSelectsRightmost(t *testing.T) {
	t.Parallel()
	m := rule.WAF{
		Enabled:        true,
		TrustProxy:     true,
		XFFIndex:       -1,
		DisableBuiltin: true,
		Deny:           []string{"198.18.0.1"},
	}
	prof, err := compileProfile(0, m)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://x/", nil)
	req.RemoteAddr = "127.0.0.1:1"
	req.Header.Set(headerXForwardedFor, "10.0.0.1, 198.18.0.1")
	if !CheckRequest(prof, req, "x", "/", http.MethodGet) {
		t.Fatal("expected deny on rightmost XFF hop")
	}
}

func TestCheckRequest_RegexOnPath(t *testing.T) {
	t.Parallel()
	m := rule.WAF{
		Enabled:        true,
		DisableBuiltin: true,
		Rules: []rule.WAFRule{{
			ID:      "re1",
			Type:    PatternTypeRegex,
			Pattern: `(?i)/admin(/|$)`,
			Targets: []string{TargetPath},
		}},
	}
	prof, err := compileProfile(0, m)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://x/Admin/extra", nil)
	req.RemoteAddr = "127.0.0.1:1"
	if !CheckRequest(prof, req, "x", "/Admin/extra", http.MethodGet) {
		t.Fatal("expected regex block on path")
	}
}

func TestCheckRequest_TargetURI_CombinesPathAndQuery(t *testing.T) {
	t.Parallel()
	m := rule.WAF{
		Enabled:        true,
		DisableBuiltin: true,
		Rules: []rule.WAFRule{{
			ID:      "u1",
			Type:    PatternTypeContains,
			Pattern: "needle",
			Targets: []string{TargetURI},
		}},
	}
	prof, err := compileProfile(0, m)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://x/p?q=needleOK", nil)
	req.RemoteAddr = "127.0.0.1:1"
	if !CheckRequest(prof, req, "x", "/p", http.MethodGet) {
		t.Fatal("expected match in query portion of synthetic URI blob")
	}
	req2 := httptest.NewRequest(http.MethodGet, "http://x/needle-in-path", nil)
	req2.RemoteAddr = "127.0.0.1:1"
	if !CheckRequest(prof, req2, "x", "/needle-in-path", http.MethodGet) {
		t.Fatal("expected match in path when no query")
	}
}

func TestCheckRequest_TargetHeaders(t *testing.T) {
	t.Parallel()
	m := rule.WAF{
		Enabled:        true,
		DisableBuiltin: true,
		Rules: []rule.WAFRule{{
			ID:      "h1",
			Type:    PatternTypeContains,
			Pattern: "evilgadget",
			Targets: []string{TargetHeaders},
		}},
	}
	prof, err := compileProfile(0, m)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://x/", nil)
	req.RemoteAddr = "127.0.0.1:1"
	req.Header.Set("User-Agent", "Mozilla/5.0 evilgadget")
	if !CheckRequest(prof, req, "x", "/", http.MethodGet) {
		t.Fatal("expected match in flattened headers blob")
	}
}

func TestCheckRequest_TargetSingleHeaderPrefix(t *testing.T) {
	t.Parallel()
	m := rule.WAF{
		Enabled:        true,
		DisableBuiltin: true,
		Rules: []rule.WAFRule{{
			ID:      "h2",
			Type:    PatternTypeContains,
			Pattern: "badref",
			Targets: []string{"header:Referer"},
		}},
	}
	prof, err := compileProfile(0, m)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://x/", nil)
	req.RemoteAddr = "127.0.0.1:1"
	req.Header.Set("Referer", "https://evil.test/badref")
	if !CheckRequest(prof, req, "x", "/", http.MethodGet) {
		t.Fatal("expected match on Referer only")
	}
}

func TestCheckRequest_BuiltinSQLi_whenNotDisabled(t *testing.T) {
	t.Parallel()
	m := rule.WAF{Enabled: true, DisableBuiltin: false}
	prof, err := compileProfile(0, m)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://x/search", nil)
	req.URL.RawQuery = "q=x union select null"
	req.RemoteAddr = "127.0.0.1:1"
	if !CheckRequest(prof, req, "x", "/search", http.MethodGet) {
		t.Fatal("expected builtin URI rule to trigger")
	}
}

func TestCheckRequest_BuiltinPathTraversal_whenNotDisabled(t *testing.T) {
	t.Parallel()
	m := rule.WAF{Enabled: true, DisableBuiltin: false}
	prof, err := compileProfile(0, m)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://x/", nil)
	req.RemoteAddr = "127.0.0.1:1"
	if !CheckRequest(prof, req, "x", "/static/../../etc/passwd", http.MethodGet) {
		t.Fatal("expected builtin path rule to trigger")
	}
}

func TestCheckRequest_TrustProxy_UnparseableXFF_UsesRemoteAddr(t *testing.T) {
	t.Parallel()
	m := rule.WAF{
		Enabled:        true,
		TrustProxy:     true,
		DisableBuiltin: true,
		Deny:           []string{"203.0.113.9"},
	}
	prof, err := compileProfile(0, m)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://x/", nil)
	req.RemoteAddr = "127.0.0.1:1"
	req.Header.Set(headerXForwardedFor, ":::not-v4-v6:::")
	if CheckRequest(prof, req, "x", "/", http.MethodGet) {
		t.Fatal("unparseable XFF must fall back to RemoteAddr")
	}
	req2 := httptest.NewRequest(http.MethodGet, "http://x/", nil)
	req2.RemoteAddr = "203.0.113.9:1"
	req2.Header.Set(headerXForwardedFor, ":::garbage:::")
	if !CheckRequest(prof, req2, "x", "/", http.MethodGet) {
		t.Fatal("fallback client must still be evaluated when XFF yields no IPs")
	}
}

func TestCheckRequest_PerRuleLogOnly_SignatureNoBlock(t *testing.T) {
	t.Parallel()
	m := rule.WAF{
		Enabled:        true,
		DisableBuiltin: true,
		Rules: []rule.WAFRule{{
			ID:      "solo",
			LogOnly: true,
			Type:    PatternTypeContains,
			Pattern: "AUDITME",
			Targets: []string{TargetQuery},
		}},
	}
	prof, err := compileProfile(0, m)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://x/p?q=AUDITME", nil)
	req.RemoteAddr = "127.0.0.1:1"
	if CheckRequest(prof, req, "x", "/p", http.MethodGet) {
		t.Fatal("per-rule log_only must not block signatures")
	}
}

func TestConcatHeaders_nilRequest(t *testing.T) {
	t.Parallel()
	if concatHeaders(nil) != "" {
		t.Fatal("nil request yields empty concatenation")
	}
}

func TestMatchBlob_regexNil_returnsFalse(t *testing.T) {
	t.Parallel()
	sr := &sigRule{contains: false, re: nil}
	if matchBlob(sr, "anything") {
		t.Fatal("nil regex")
	}
}

func TestMatchesSignature_unknownTargetKind_skipped(t *testing.T) {
	t.Parallel()
	sr := &sigRule{id: "x", targets: []targetKind{targetKind(99)}, hdrNames: []string{""}}
	req := httptest.NewRequest(http.MethodGet, "http://example/p", nil)
	if matchesSignature(sr, req, "/p", "q=n") {
		t.Fatal("unknown kind should never match blob")
	}
}

func TestClientIP_remoteAddrBareIP_noPort(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest(http.MethodGet, "http://x/", nil)
	r.RemoteAddr = "198.51.100.51"
	got := clientIP(r, false, 0)
	if got == nil || got.String() != "198.51.100.51" {
		t.Fatalf("got %v", got)
	}
}

func TestClientIP_trustQuotedXFF_and_outOfRangeFallsBack(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest(http.MethodGet, "http://x/", nil)
	r.RemoteAddr = "127.0.0.1:9"
	r.Header.Set(headerXForwardedFor, `"203.0.113.61", unknown, 203.0.113.61`)
	got := clientIP(r, true, 0).String()
	if got != "203.0.113.61" {
		t.Fatalf("first hop %q", got)
	}

	r2 := httptest.NewRequest(http.MethodGet, "http://x/", nil)
	r2.RemoteAddr = "198.51.100.61:444"
	r2.Header.Set(headerXForwardedFor, "203.0.113.71")
	got2 := clientIP(r2, true, 5).String()
	if got2 != "198.51.100.61" {
		t.Fatalf("OOB idx should fallback to RemoteAddr, got %q", got2)
	}

	r3 := httptest.NewRequest(http.MethodGet, "http://x/", nil)
	r3.RemoteAddr = "10.44.44.44:1"
	got3 := clientIP(r3, true, 0).String()
	if got3 != "10.44.44.44" {
		t.Fatalf("empty XFF uses direct addr: %q", got3)
	}
}

func TestClientIP_XFF_emptyChunks_skipped(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest(http.MethodGet, "http://x/", nil)
	r.RemoteAddr = "10.0.0.2:1"
	r.Header.Set(headerXForwardedFor, " , , 10.0.0.1 ")
	if clientIP(r, true, 0).String() != "10.0.0.1" {
		t.Fatal("empty XFF segments should be skipped")
	}
}

func TestMatchesSignature_tkHeader_indexBounds(t *testing.T) {
	t.Parallel()
	sr := &sigRule{
		id:       "hdr",
		contains: true,
		pattern:  "secret",
		targets:  []targetKind{tkHeader},
		hdrNames: []string{},
	}
	req := httptest.NewRequest(http.MethodGet, "http://x/", nil)
	req.Header.Set("X-Test", "secret")
	if matchesSignature(sr, req, "/", "") {
		t.Fatal("empty hdrNames at index should not read wrong header")
	}
}
