package waf

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-zoox/ingress/core/rule"
)

func TestCompileProfile_InvalidDeny(t *testing.T) {
	t.Parallel()
	_, err := compileProfile(0, rule.WAF{Enabled: true, DisableBuiltin: true, Deny: []string{"not-an-ip"}})
	if err == nil {
		t.Fatal("expected error for invalid deny entry")
	}
}

func TestCompileProfile_InvalidRegex(t *testing.T) {
	t.Parallel()
	_, err := compileProfile(0, rule.WAF{
		Enabled:        true,
		DisableBuiltin: true,
		Rules: []rule.WAFRule{{
			ID:      "badre",
			Type:    PatternTypeRegex,
			Pattern: `(?b`,
			Targets: []string{TargetPath},
		}},
	})
	if err == nil {
		t.Fatal("expected invalid regex error")
	}
}

func TestCompileProfile_DuplicateRuleID_AfterBuiltinMerge(t *testing.T) {
	t.Parallel()
	_, err := compileProfile(0, rule.WAF{
		Enabled:        true,
		DisableBuiltin: false,
		Rules: []rule.WAFRule{{
			ID:      "builtin:sqli-common",
			Type:    PatternTypeContains,
			Pattern: "x",
			Targets: []string{TargetPath},
		}},
	})
	if err == nil {
		t.Fatal("expected duplicate id after starter merge")
	}
}

func TestCompileProfile_EmptyTargets(t *testing.T) {
	t.Parallel()
	_, err := compileProfile(0, rule.WAF{
		Enabled:        true,
		DisableBuiltin: true,
		Rules: []rule.WAFRule{{
			ID:      "e1",
			Type:    PatternTypeContains,
			Pattern: "x",
			Targets: nil,
		}},
	})
	if err == nil {
		t.Fatal("expected targets required error")
	}
}

func TestCompileIngress_PerRuleAndFallback(t *testing.T) {
	t.Parallel()
	global := rule.WAF{Enabled: true, DisableBuiltin: true, Deny: []string{"192.0.2.1"}}
	rules := []rule.Rule{
		{},
		{WAFPatch: map[string]any{"deny": []any{"192.0.2.2"}}},
	}
	per, fb, err := CompileIngress(global, rules)
	if err != nil {
		t.Fatal(err)
	}
	if len(per) != 2 || fb == nil {
		t.Fatalf("profiles: per=%d fb=%v", len(per), fb)
	}
	req := httptest.NewRequest(http.MethodGet, "http://h/", nil)
	req.RemoteAddr = "192.0.2.1:1"
	if !CheckRequest(per[0], req, "h", "/", http.MethodGet, nil) {
		t.Fatal("rule0 should inherit global deny")
	}
	if CheckRequest(per[1], req, "h", "/", http.MethodGet, nil) {
		t.Fatal("rule1 patch should replace deny list")
	}
	req2 := httptest.NewRequest(http.MethodGet, "http://h/", nil)
	req2.RemoteAddr = "192.0.2.2:1"
	if !CheckRequest(per[1], req2, "h", "/", http.MethodGet, nil) {
		t.Fatal("rule1 should use patched deny")
	}
	reqFB := httptest.NewRequest(http.MethodGet, "http://h/", nil)
	reqFB.RemoteAddr = "192.0.2.1:1"
	if !CheckRequest(fb, reqFB, "h", "/", http.MethodGet, nil) {
		t.Fatal("fallback profile should reflect global-only merge")
	}
}

func TestCompileProfile_CustomBlockResponseFields(t *testing.T) {
	t.Parallel()
	p, err := compileProfile(0, rule.WAF{
		Enabled: true, DisableBuiltin: true,
		BlockStatusCode: 429, BlockContentType: "application/json", BlockBody: `{"ok":false}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.BlockStatus != 429 || p.BlockContentType != "application/json" || p.BlockBody != `{"ok":false}` {
		t.Fatalf("unexpected block response: %d %q %q", p.BlockStatus, p.BlockContentType, p.BlockBody)
	}
}

func TestCompileProfile_UnsupportedPatternType(t *testing.T) {
	t.Parallel()
	_, err := compileProfile(0, rule.WAF{
		Enabled: true, DisableBuiltin: true,
		Rules: []rule.WAFRule{{
			ID: "badt", Type: "fuzzy", Pattern: "x", Targets: []string{TargetPath},
		}},
	})
	if err == nil {
		t.Fatal("expected unsupported type error")
	}
}

func TestCompileProfile_UnknownTarget(t *testing.T) {
	t.Parallel()
	_, err := compileProfile(0, rule.WAF{
		Enabled: true, DisableBuiltin: true,
		Rules: []rule.WAFRule{{
			ID: "badtg", Type: PatternTypeContains, Pattern: "x", Targets: []string{"request_body"},
		}},
	})
	if err == nil {
		t.Fatal("expected unknown targets error")
	}
}

func TestCompileProfile_EmptyRuleID(t *testing.T) {
	t.Parallel()
	_, err := compileProfile(0, rule.WAF{
		Enabled: true, DisableBuiltin: true,
		Rules: []rule.WAFRule{{
			ID: "", Type: PatternTypeContains, Pattern: "x", Targets: []string{TargetPath},
		}},
	})
	if err == nil {
		t.Fatal("expected id required error")
	}
}

func TestCompileProfile_DuplicateCustomRuleIDs(t *testing.T) {
	t.Parallel()
	_, err := compileProfile(0, rule.WAF{
		Enabled: true, DisableBuiltin: true,
		Rules: []rule.WAFRule{
			{ID: "dup", Type: PatternTypeContains, Pattern: "a", Targets: []string{TargetPath}},
			{ID: "dup", Type: PatternTypeContains, Pattern: "b", Targets: []string{TargetPath}},
		},
	})
	if err == nil {
		t.Fatal("expected duplicate custom id error")
	}
}

func TestCompileIngress_MergePatchError(t *testing.T) {
	t.Parallel()
	_, _, err := CompileIngress(rule.WAF{}, []rule.Rule{
		{WAFPatch: map[string]any{"enabled": "?"}},
	})
	if err == nil {
		t.Fatal("expected merge error propagates")
	}
}

func TestCompileIngress_CompileProfileError(t *testing.T) {
	t.Parallel()
	_, _, err := CompileIngress(rule.WAF{Enabled: true, Deny: []string{"not-ip"}}, []rule.Rule{{}})
	if err == nil {
		t.Fatal("expected compile deny error")
	}
}

func TestCompileProfile_denies_TrimsWhitespaceAndSkipsEmpty(t *testing.T) {
	t.Parallel()
	p, err := compileProfile(-1, rule.WAF{
		Enabled: true, DisableBuiltin: true,
		Deny: []string{"  192.0.2.4  ", ""},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.denyNet) != 1 {
		t.Fatalf("deny nets: %d", len(p.denyNet))
	}
}

func TestCompileProfile_Allow_InvalidIP(t *testing.T) {
	t.Parallel()
	_, err := compileProfile(-1, rule.WAF{Enabled: true, DisableBuiltin: true, Allow: []string{"nope"}})
	if err == nil || !strings.Contains(err.Error(), "waf(global).waf.allow") {
		t.Fatalf("err=%v", err)
	}
}

func TestCompileProfile_BlockContentType_whitespaceKeepsDefault(t *testing.T) {
	t.Parallel()
	p, err := compileProfile(0, rule.WAF{Enabled: true, DisableBuiltin: true, BlockContentType: " \t"})
	if err != nil {
		t.Fatal(err)
	}
	if p.BlockContentType != "text/plain; charset=utf-8" {
		t.Fatalf("got %q", p.BlockContentType)
	}
}

func TestCompileProfile_EmptyPatternAfterTrim(t *testing.T) {
	t.Parallel()
	_, err := compileProfile(2, rule.WAF{
		Enabled: true, DisableBuiltin: true,
		Rules: []rule.WAFRule{{ID: "e", Pattern: "  \t", Targets: []string{TargetPath}}},
	})
	if err == nil || !strings.Contains(err.Error(), "pattern is required") {
		t.Fatalf("err=%v", err)
	}
}

func TestCompileSigRule_TargetEmptyEntry(t *testing.T) {
	t.Parallel()
	_, err := compileProfile(1, rule.WAF{
		Enabled: true, DisableBuiltin: true,
		Rules: []rule.WAFRule{{
			ID: "tg", Pattern: ".", Targets: []string{TargetPath, "  ", TargetQuery}},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "empty entry") {
		t.Fatalf("err=%v", err)
	}
}

func TestCompileSigRule_TargetHeaderMissingName(t *testing.T) {
	t.Parallel()
	_, err := compileProfile(0, rule.WAF{
		Enabled: true, DisableBuiltin: true,
		Rules: []rule.WAFRule{{
			ID:      "tg2",
			Pattern: ".",
			Targets: []string{"header:\t"},
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "empty header name") {
		t.Fatalf("err=%v", err)
	}
}

func TestParseIPCIDR_V6_plain(t *testing.T) {
	t.Parallel()
	n, err := parseIPCIDR("2001:db8::42")
	if err != nil {
		t.Fatal(err)
	}
	ip := net.ParseIP("2001:db8::42")
	if !n.Contains(ip) {
		t.Fatal("v6 net should contain parsed ip")
	}
}

func TestParseIPCIDR_InvalidCIDR_WithSlash(t *testing.T) {
	t.Parallel()
	_, err := parseIPCIDR("2001:db8::/129")
	if err == nil {
		t.Fatal("expected invalid CIDR")
	}
}

func TestCompileProfile_IPv6_deny_matches(t *testing.T) {
	t.Parallel()
	ip := net.ParseIP("fd00::abcd")
	raw := strings.ReplaceAll(strings.Trim(ip.String(), "[]"), "%", "")
	m := rule.WAF{Enabled: true, DisableBuiltin: true, Deny: []string{raw}}
	prof, err := compileProfile(0, m)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://x/", nil)
	req.RemoteAddr = "[" + raw + "]:1234"
	if !CheckRequest(prof, req, "x", "/", http.MethodGet, nil) {
		t.Fatal("ipv6 deny should trigger")
	}
}

func TestIPMatchesNets_nilIP(t *testing.T) {
	t.Parallel()
	_, ipnet, _ := net.ParseCIDR("10.10.10.10/32")
	if ipMatchesNets(nil, []*net.IPNet{ipnet}) {
		t.Fatal("nil ip")
	}
}

func TestIPMatchesNets_nilNetEntry(t *testing.T) {
	t.Parallel()
	ip := net.ParseIP("10.0.0.1")
	var extra *net.IPNet
	_, classA, err := net.ParseCIDR("10.0.0.0/8")
	if err != nil {
		t.Fatal(err)
	}
	nets := []*net.IPNet{nil, extra, classA}
	if !ipMatchesNets(ip, nets) {
		t.Fatal("should match usable net, skipping nil entries")
	}
}
