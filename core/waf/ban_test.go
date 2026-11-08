package waf

import (
	"net"
	"net/http/httptest"
	"testing"
	"time"
)

func mustCompileBans(t *testing.T, entries []BanEntry) *BanSet {
	t.Helper()
	set, err := CompileBans(entries, nil)
	if err != nil {
		t.Fatalf("CompileBans: %v", err)
	}
	return set
}

func TestCompileBans_Kinds(t *testing.T) {
	cases := []struct {
		name    string
		entry   BanEntry
		wantErr bool
	}{
		{name: "ip v4", entry: BanEntry{Kind: BanKindIP, Value: "203.0.113.42"}},
		{name: "ip v6", entry: BanEntry{Kind: BanKindIP, Value: "2001:db8::1"}},
		{name: "ip invalid", entry: BanEntry{Kind: BanKindIP, Value: "999.1.1.1"}, wantErr: true},
		{name: "cidr v4", entry: BanEntry{Kind: BanKindIPCIDR, Value: "203.0.113.0/24"}},
		{name: "cidr v6", entry: BanEntry{Kind: BanKindIPCIDR, Value: "2001:db8::/32"}},
		{name: "cidr invalid", entry: BanEntry{Kind: BanKindIPCIDR, Value: "203.0.113.0/99"}, wantErr: true},
		{name: "ua contains", entry: BanEntry{Kind: BanKindUAContains, Value: "sqlmap"}},
		{name: "ua regex", entry: BanEntry{Kind: BanKindUARegex, Value: `(?i)(nikto|masscan)`}},
		{name: "ua regex invalid", entry: BanEntry{Kind: BanKindUARegex, Value: `(?i`}, wantErr: true},
		{name: "empty value", entry: BanEntry{Kind: BanKindIP, Value: "  "}, wantErr: true},
		{name: "unknown kind", entry: BanEntry{Kind: "nope", Value: "x"}, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := CompileBans([]BanEntry{tc.entry}, nil)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestCompileBans_EmptyKindDefaultsToIP(t *testing.T) {
	set := mustCompileBans(t, []BanEntry{{Kind: "", Value: "203.0.113.42"}})
	hit := set.CheckWithClientIP(net.ParseIP("203.0.113.42"), "example.com", "", time.Now())
	if hit == nil {
		t.Fatalf("expected hit for default ip kind")
	}
	if hit.Label != "ban:ip" {
		t.Fatalf("label = %q, want ban:ip", hit.Label)
	}
}

func TestBanSet_IPv4IPv6Normalization(t *testing.T) {
	// A ban registered as plain IPv4 must also match the IPv4-mapped IPv6 form
	// the TCP stack may report (::ffff:203.0.113.42).
	set := mustCompileBans(t, []BanEntry{{ID: 1, Kind: BanKindIP, Value: "203.0.113.42"}})

	for _, raw := range []string{"203.0.113.42", "::ffff:203.0.113.42"} {
		hit := set.CheckWithClientIP(net.ParseIP(raw), "example.com", "", time.Now())
		if hit == nil {
			t.Fatalf("expected hit for %q", raw)
		}
		if hit.ID != 1 {
			t.Fatalf("hit id = %d, want 1", hit.ID)
		}
	}

	if hit := set.CheckWithClientIP(net.ParseIP("203.0.113.43"), "example.com", "", time.Now()); hit != nil {
		t.Fatalf("expected no hit for different IP, got %+v", hit)
	}
}

func TestBanSet_CIDR(t *testing.T) {
	set := mustCompileBans(t, []BanEntry{{ID: 7, Kind: BanKindIPCIDR, Value: "198.51.100.0/24"}})

	if hit := set.CheckWithClientIP(net.ParseIP("198.51.100.77"), "example.com", "", time.Now()); hit == nil || hit.ID != 7 {
		t.Fatalf("expected CIDR hit, got %+v", hit)
	}
	if hit := set.CheckWithClientIP(net.ParseIP("198.51.101.77"), "example.com", "", time.Now()); hit != nil {
		t.Fatalf("expected no hit outside CIDR, got %+v", hit)
	}
}

func TestBanSet_UAContainsCaseInsensitive(t *testing.T) {
	set := mustCompileBans(t, []BanEntry{{ID: 3, Kind: BanKindUAContains, Value: "SQLMap"}})

	for _, ua := range []string{"sqlmap/1.7.2#stable", "SQLMAP", "Mozilla/5.0 (compatible; SQLMap)"} {
		if hit := set.CheckWithClientIP(nil, "example.com", ua, time.Now()); hit == nil || hit.ID != 3 {
			t.Fatalf("expected UA contains hit for %q, got %+v", ua, hit)
		}
	}
	if hit := set.CheckWithClientIP(nil, "example.com", "curl/8.4.0", time.Now()); hit != nil {
		t.Fatalf("expected no hit, got %+v", hit)
	}
}

func TestBanSet_UARegex(t *testing.T) {
	set := mustCompileBans(t, []BanEntry{{ID: 4, Kind: BanKindUARegex, Value: `(?i)(nikto|masscan|gobuster)`}})

	if hit := set.CheckWithClientIP(nil, "example.com", "Nikto/2.5.0", time.Now()); hit == nil || hit.ID != 4 {
		t.Fatalf("expected UA regex hit, got %+v", hit)
	}
	if hit := set.CheckWithClientIP(nil, "example.com", "curl/8.4.0", time.Now()); hit != nil {
		t.Fatalf("expected no hit, got %+v", hit)
	}
	// Empty UA must not match a regex that would otherwise match empty string.
	emptyRe := mustCompileBans(t, []BanEntry{{ID: 5, Kind: BanKindUARegex, Value: `.*`}})
	if hit := emptyRe.CheckWithClientIP(nil, "example.com", "", time.Now()); hit != nil {
		t.Fatalf("empty UA should not match, got %+v", hit)
	}
}

func TestBanSet_HostScope(t *testing.T) {
	set := mustCompileBans(t, []BanEntry{
		{ID: 10, Kind: BanKindIP, Value: "203.0.113.42", Hosts: []string{"api.example.com"}},
	})

	now := time.Now()
	ip := net.ParseIP("203.0.113.42")

	if hit := set.CheckWithClientIP(ip, "api.example.com", "", now); hit == nil || hit.ID != 10 {
		t.Fatalf("expected hit on scoped host, got %+v", hit)
	}
	if hit := set.CheckWithClientIP(ip, "www.example.com", "", now); hit != nil {
		t.Fatalf("expected no hit on other host, got %+v", hit)
	}
}

func TestBanSet_HostScopeWildcard(t *testing.T) {
	set := mustCompileBans(t, []BanEntry{
		{ID: 11, Kind: BanKindIP, Value: "203.0.113.42", Hosts: []string{"*.example.com"}},
	})

	now := time.Now()
	ip := net.ParseIP("203.0.113.42")

	if hit := set.CheckWithClientIP(ip, "api.example.com", "", now); hit == nil {
		t.Fatalf("expected wildcard host hit")
	}
	if hit := set.CheckWithClientIP(ip, "example.org", "", now); hit != nil {
		t.Fatalf("expected no hit for other domain, got %+v", hit)
	}
}

func TestBanSet_EmptyHostsIsGlobal(t *testing.T) {
	set := mustCompileBans(t, []BanEntry{{ID: 12, Kind: BanKindIP, Value: "203.0.113.42"}})

	for _, host := range []string{"a.example.com", "b.example.org", ""} {
		if hit := set.CheckWithClientIP(net.ParseIP("203.0.113.42"), host, "", time.Now()); hit == nil {
			t.Fatalf("expected global hit for host %q", host)
		}
	}
}

func TestBanSet_TTLExpiry(t *testing.T) {
	now := time.Now()
	set := mustCompileBans(t, []BanEntry{
		{ID: 20, Kind: BanKindIP, Value: "203.0.113.42", ExpiresAt: now.Add(-time.Minute)},
		{ID: 21, Kind: BanKindIP, Value: "203.0.113.43", ExpiresAt: now.Add(time.Hour)},
		{ID: 22, Kind: BanKindIP, Value: "203.0.113.44"}, // permanent
	})

	ip42 := net.ParseIP("203.0.113.42")
	if hit := set.CheckWithClientIP(ip42, "example.com", "", now); hit != nil {
		t.Fatalf("expired entry must not match, got %+v", hit)
	}
	ip43 := net.ParseIP("203.0.113.43")
	if hit := set.CheckWithClientIP(ip43, "example.com", "", now); hit == nil {
		t.Fatalf("not-yet-expired entry must match")
	}
	ip44 := net.ParseIP("203.0.113.44")
	if hit := set.CheckWithClientIP(ip44, "example.com", "", now.Add(24*time.Hour)); hit == nil {
		t.Fatalf("permanent entry must always match")
	}

	if got := set.ActiveEntryCount(now); got != 2 {
		t.Fatalf("ActiveEntryCount = %d, want 2", got)
	}
	if got := set.Len(); got != 3 {
		t.Fatalf("Len = %d, want 3", got)
	}
}

func TestBanSet_NilSafety(t *testing.T) {
	var set *BanSet
	if hit := set.CheckWithClientIP(net.ParseIP("1.2.3.4"), "h", "ua", time.Now()); hit != nil {
		t.Fatalf("nil set must not match")
	}
	if got := set.Len(); got != 0 {
		t.Fatalf("nil set Len = %d, want 0", got)
	}
	if got := set.ActiveEntryCount(time.Now()); got != 0 {
		t.Fatalf("nil set ActiveEntryCount = %d, want 0", got)
	}
	if hit := set.Check(nil, "h"); hit != nil {
		t.Fatalf("nil request must not match")
	}
}

func TestBanSet_CheckUsesPeerIP(t *testing.T) {
	set := mustCompileBans(t, []BanEntry{{ID: 30, Kind: BanKindIP, Value: "203.0.113.42"}})

	req := httptest.NewRequest("GET", "http://api.example.com/x", nil)
	req.RemoteAddr = "203.0.113.42:12345"

	hit := set.Check(req, "api.example.com")
	if hit == nil {
		t.Fatalf("expected hit from RemoteAddr peer")
	}
	if hit.ClientIP != "203.0.113.42" {
		t.Fatalf("ClientIP = %q, want 203.0.113.42", hit.ClientIP)
	}
}

func TestBanSet_CheckIgnoresSpoofedXFF(t *testing.T) {
	// Without trusted_proxies the forged X-Forwarded-For must be ignored.
	set := mustCompileBans(t, []BanEntry{{ID: 31, Kind: BanKindIP, Value: "203.0.113.42"}})

	req := httptest.NewRequest("GET", "http://api.example.com/x", nil)
	req.RemoteAddr = "198.51.100.9:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.42")

	if hit := set.Check(req, "api.example.com"); hit != nil {
		t.Fatalf("forged XFF must not ban the request, got %+v", hit)
	}
}

func TestBanSet_TrustedProxyResolution(t *testing.T) {
	trusted, err := ParseTrustedProxyNets([]string{"10.0.0.0/8"})
	if err != nil {
		t.Fatalf("ParseTrustedProxyNets: %v", err)
	}
	set, err := CompileBans([]BanEntry{{ID: 32, Kind: BanKindIP, Value: "203.0.113.42"}}, trusted)
	if err != nil {
		t.Fatalf("CompileBans: %v", err)
	}

	req := httptest.NewRequest("GET", "http://api.example.com/x", nil)
	req.RemoteAddr = "10.1.2.3:12345" // trusted proxy
	req.Header.Set("X-Forwarded-For", "203.0.113.42")

	hit := set.Check(req, "api.example.com")
	if hit == nil {
		t.Fatalf("expected hit via trusted proxy XFF")
	}
	if hit.ClientIP != "203.0.113.42" {
		t.Fatalf("ClientIP = %q, want the XFF hop", hit.ClientIP)
	}
}

func TestValidateBanValue(t *testing.T) {
	if err := ValidateBanValue(BanKindIP, "203.0.113.42"); err != nil {
		t.Fatalf("valid IP rejected: %v", err)
	}
	if err := ValidateBanValue(BanKindIP, "not-an-ip"); err == nil {
		t.Fatalf("invalid IP accepted")
	}
	if err := ValidateBanValue(BanKindUARegex, `(?i)sqlmap`); err != nil {
		t.Fatalf("valid regex rejected: %v", err)
	}
	if err := ValidateBanValue(BanKindUARegex, `(?i`); err == nil {
		t.Fatalf("invalid regex accepted")
	}
}

func TestBanSet_PriorityOrder(t *testing.T) {
	// Exact IP should be reported before a CIDR that also matches, so the
	// most specific ban is what shows up on the event.
	set := mustCompileBans(t, []BanEntry{
		{ID: 40, Kind: BanKindIPCIDR, Value: "203.0.113.0/24"},
		{ID: 41, Kind: BanKindIP, Value: "203.0.113.42"},
	})

	hit := set.CheckWithClientIP(net.ParseIP("203.0.113.42"), "example.com", "", time.Now())
	if hit == nil || hit.ID != 41 {
		t.Fatalf("expected exact IP ban (41) to win, got %+v", hit)
	}
}

func BenchmarkBanSetCheck(b *testing.B) {
	entries := make([]BanEntry, 0, 1150)
	for i := 0; i < 1000; i++ {
		entries = append(entries, BanEntry{
			ID:    uint(i + 1),
			Kind:  BanKindIP,
			Value: net.IPv4(203, 0, byte(i/256), byte(i%256)).String(),
		})
	}
	for i := 0; i < 100; i++ {
		entries = append(entries, BanEntry{
			ID:    uint(2000 + i),
			Kind:  BanKindIPCIDR,
			Value: net.IPv4(198, 51, byte(i), 0).String() + "/24",
		})
	}
	for i := 0; i < 50; i++ {
		entries = append(entries, BanEntry{
			ID:    uint(3000 + i),
			Kind:  BanKindUAContains,
			Value: "scanner-" + string(rune('a'+i%26)) + "-agent",
		})
	}

	set, err := CompileBans(entries, nil)
	if err != nil {
		b.Fatalf("CompileBans: %v", err)
	}
	ip := net.ParseIP("192.0.2.10")
	ua := "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36"
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = set.CheckWithClientIP(ip, "api.example.com", ua, now)
	}
}
