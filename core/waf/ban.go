package waf

import (
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// BanKind is the runtime ban entry type.
type BanKind string

const (
	BanKindIP         BanKind = "ip"
	BanKindIPCIDR     BanKind = "ip_cidr"
	BanKindUAContains BanKind = "ua_contains"
	BanKindUARegex    BanKind = "ua_regex"
)

// BanSource records who created a runtime ban entry.
const (
	BanSourceManual = "manual"
	BanSourceAuto   = "auto"
)

// BanEntry is one runtime ban rule as compiled input (mirrors the admin DB row).
//
// ExpiresAt zero value means "permanent". Expiry is evaluated at check time, not
// at compile time, so a ban stops matching the moment it expires without any
// recompile or background job.
type BanEntry struct {
	ID        uint
	Kind      BanKind
	Value     string
	Hosts     []string
	ExpiresAt time.Time
	Reason    string
}

// Label returns the human-readable rule name recorded on WAF events.
func (e BanEntry) Label() string {
	kind := string(e.Kind)
	if kind == "" {
		kind = string(BanKindIP)
	}
	return "ban:" + kind
}

// compiledBan is one ban entry with all matching state pre-compiled.
type compiledBan struct {
	id        uint
	kind      BanKind
	label     string
	value     string
	hosts     HostPatternList
	expiresAt time.Time

	// IP kinds
	ip    net.IP
	ipNet *net.IPNet

	// UA kinds
	uaContains string
	uaRe       *regexp.Regexp
}

// expired reports whether the entry no longer applies at now.
func (b *compiledBan) expired(now time.Time) bool {
	return !b.expiresAt.IsZero() && !now.Before(b.expiresAt)
}

// matchIP reports whether ip matches this entry (IP kinds only).
func (b *compiledBan) matchIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	switch b.kind {
	case BanKindIP:
		if b.ip == nil {
			return false
		}
		return b.ip.Equal(ip)
	case BanKindIPCIDR:
		if b.ipNet == nil {
			return false
		}
		return b.ipNet.Contains(ip)
	default:
		return false
	}
}

// matchUA reports whether userAgent matches this entry (UA kinds only).
// lowerUA is the pre-lowercased user agent, computed once per request by the
// caller: lowercasing per rule would dominate the scan for large ban lists.
func (b *compiledBan) matchUA(userAgent, lowerUA string) bool {
	switch b.kind {
	case BanKindUAContains:
		return b.uaContains != "" && strings.Contains(lowerUA, b.uaContains)
	case BanKindUARegex:
		return b.uaRe != nil && b.uaRe.MatchString(userAgent)
	default:
		return false
	}
}

// matchHost reports whether the entry applies to hostname (empty Hosts = global).
func (b *compiledBan) matchHost(hostname string) bool {
	if b.hosts.Len() == 0 {
		return true
	}
	return b.hosts.Matches(hostname)
}

// BanHit describes one matched ban entry.
type BanHit struct {
	ID    uint
	Kind  BanKind
	Value string
	Label string
	// ClientIP is the resolved client address used for the match (may be empty
	// when the peer address could not be parsed).
	ClientIP string
}

// BanSet is an immutable compiled snapshot of the runtime ban list.
//
// It is replaced wholesale on every change (atomic pointer swap on the core), so
// checks are lock-free and readers always observe a consistent set.
type BanSet struct {
	ipExact map[string][]*compiledBan
	ipNets  []*compiledBan
	uaRules []*compiledBan

	// trustedNets are the proxy CIDRs used to resolve the client IP, mirroring
	// the WAF client IP algorithm.
	trustedNets []*net.IPNet

	entryCount int
}

// normalizeBanIP normalizes an IP so that ::ffff:1.2.3.4 and 1.2.3.4 compare equal.
func normalizeBanIP(ip net.IP) net.IP {
	if ip == nil {
		return nil
	}
	if v4 := ip.To4(); v4 != nil {
		return v4
	}
	return ip.To16()
}

// CompileBans builds an immutable ban snapshot.
//
// trustedNets are the proxy CIDRs (waf.trusted_proxies) used for client IP
// resolution; pass nil to use the raw TCP peer only.
func CompileBans(entries []BanEntry, trustedNets []*net.IPNet) (*BanSet, error) {
	set := &BanSet{
		ipExact:     make(map[string][]*compiledBan),
		trustedNets: trustedNets,
	}

	for i, entry := range entries {
		cb, err := compileBanEntry(entry)
		if err != nil {
			return nil, fmt.Errorf("ban[%d] id=%d kind=%q value=%q: %w", i, entry.ID, entry.Kind, entry.Value, err)
		}
		switch cb.kind {
		case BanKindIP:
			key := cb.ip.String()
			set.ipExact[key] = append(set.ipExact[key], cb)
		case BanKindIPCIDR:
			set.ipNets = append(set.ipNets, cb)
		case BanKindUAContains, BanKindUARegex:
			set.uaRules = append(set.uaRules, cb)
		}
		set.entryCount++
	}

	return set, nil
}

func compileBanEntry(entry BanEntry) (*compiledBan, error) {
	kind := BanKind(strings.TrimSpace(strings.ToLower(string(entry.Kind))))
	if kind == "" {
		kind = BanKindIP
	}

	value := strings.TrimSpace(entry.Value)
	if value == "" {
		return nil, fmt.Errorf("value is required")
	}

	cb := &compiledBan{
		id:        entry.ID,
		kind:      kind,
		value:     value,
		expiresAt: entry.ExpiresAt,
	}
	cb.label = BanEntry{Kind: kind}.Label()

	// Host scope: empty means global.
	if len(entry.Hosts) > 0 {
		hosts, err := CompileHostPatternList(entry.Hosts, "hosts")
		if err != nil {
			return nil, err
		}
		cb.hosts = hosts
	}

	switch kind {
	case BanKindIP:
		ip := net.ParseIP(value)
		if ip == nil {
			return nil, fmt.Errorf("invalid IP %q", value)
		}
		cb.ip = normalizeBanIP(ip)

	case BanKindIPCIDR:
		_, ipNet, err := net.ParseCIDR(value)
		if err != nil {
			return nil, fmt.Errorf("invalid CIDR %q: %w", value, err)
		}
		cb.ipNet = ipNet

	case BanKindUAContains:
		cb.uaContains = strings.ToLower(value)

	case BanKindUARegex:
		re, err := regexp.Compile(value)
		if err != nil {
			return nil, fmt.Errorf("invalid regex %q: %w", value, err)
		}
		cb.uaRe = re

	default:
		return nil, fmt.Errorf("unsupported kind %q (use %q, %q, %q, or %q)",
			kind, BanKindIP, BanKindIPCIDR, BanKindUAContains, BanKindUARegex)
	}

	return cb, nil
}

// Len reports how many entries the set holds (including expired ones).
func (s *BanSet) Len() int {
	if s == nil {
		return 0
	}
	return s.entryCount
}

// Check evaluates the ban set for one request.
//
// It returns the matched entry, or nil when the request is not banned. The
// client IP is resolved with the same algorithm the WAF uses (real TCP peer
// first, then trusted_proxies), so bans and WAF IP rules never disagree.
func (s *BanSet) Check(r *http.Request, hostname string) *BanHit {
	if s == nil || r == nil {
		return nil
	}

	ip := clientIP(r, false, 0, s.trustedNets)
	hit := s.CheckWithClientIP(ip, hostname, r.UserAgent(), time.Now())
	if hit != nil && ip != nil {
		hit.ClientIP = ip.String()
	}
	return hit
}

// CheckWithClientIP is the pure evaluation core; time is injectable for tests.
func (s *BanSet) CheckWithClientIP(ip net.IP, hostname, userAgent string, now time.Time) *BanHit {
	if s == nil {
		return nil
	}
	ip = normalizeBanIP(ip)

	// Exact IP matches are the common case and cheapest: check them first.
	if ip != nil {
		if entries, ok := s.ipExact[ip.String()]; ok {
			for _, cb := range entries {
				if cb.expired(now) || !cb.matchHost(hostname) {
					continue
				}
				return &BanHit{ID: cb.id, Kind: cb.kind, Value: cb.value, Label: cb.label}
			}
		}
	}

	for _, cb := range s.ipNets {
		if cb.expired(now) || !cb.matchHost(hostname) {
			continue
		}
		if cb.matchIP(ip) {
			return &BanHit{ID: cb.id, Kind: cb.kind, Value: cb.value, Label: cb.label}
		}
	}

	if userAgent != "" && len(s.uaRules) > 0 {
		lowerUA := strings.ToLower(userAgent)
		for _, cb := range s.uaRules {
			if cb.expired(now) || !cb.matchHost(hostname) {
				continue
			}
			if cb.matchUA(userAgent, lowerUA) {
				return &BanHit{ID: cb.id, Kind: cb.kind, Value: cb.value, Label: cb.label}
			}
		}
	}

	return nil
}

// ActiveEntryCount reports how many entries are still active at now.
func (s *BanSet) ActiveEntryCount(now time.Time) int {
	if s == nil {
		return 0
	}
	n := 0
	for _, entries := range s.ipExact {
		for _, cb := range entries {
			if !cb.expired(now) {
				n++
			}
		}
	}
	for _, cb := range s.ipNets {
		if !cb.expired(now) {
			n++
		}
	}
	for _, cb := range s.uaRules {
		if !cb.expired(now) {
			n++
		}
	}
	return n
}

// ValidateBanValue checks a ban value without compiling a full set. It is
// exported so the admin API can reject bad input before touching the database.
func ValidateBanValue(kind BanKind, value string) error {
	_, err := compileBanEntry(BanEntry{Kind: kind, Value: value})
	return err
}

// CompileBanHosts validates ban host patterns without building a full set.
// It is exported so the admin API can reject malformed host scope early.
func CompileBanHosts(hosts []string) (HostPatternList, error) {
	return CompileHostPatternList(hosts, "ban.hosts")
}

// ParseTrustedProxyNets parses trusted proxy CIDRs for ban client IP resolution.
// It reuses the same rules as WAF (invalid entries and /0 are rejected).
func ParseTrustedProxyNets(raw []string) ([]*net.IPNet, error) {
	return compileTrustedProxyNets(raw, "waf.trusted_proxies")
}
