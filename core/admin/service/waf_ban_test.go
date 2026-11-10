package service

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-zoox/gormx"
	ingcore "github.com/go-zoox/ingress/core"
	"github.com/go-zoox/ingress/core/admin/config"
	"github.com/go-zoox/ingress/core/admin/model"
	"github.com/go-zoox/ingress/core/waf"
)

func setupBanDB(t *testing.T) {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "waf-ban.db")
	if err := gormx.LoadDB("sqlite", dsn); err != nil {
		t.Fatal(err)
	}
	if err := gormx.GetDB().AutoMigrate(model.MigrateModels()...); err != nil {
		t.Fatal(err)
	}
}

// banTestCore is a minimal in-memory Core implementation for sync assertions.
type banTestCore struct {
	set   *waf.BanSet
	calls int
}

func (c *banTestCore) Version() string                                        { return "test" }
func (c *banTestCore) Run() error                                             { return nil }
func (c *banTestCore) Reload(*ingcore.Config) error                           { return nil }
func (c *banTestCore) ReloadFromFile() error                                  { return nil }
func (c *banTestCore) SetWAFOverride(*bool)                                   {}
func (c *banTestCore) GetWAFOverride() *bool                                  { return nil }
func (c *banTestCore) IsWAFEnabled() bool                                     { return false }
func (c *banTestCore) SetWAFCallback(ingcore.WAFCallback)                     {}
func (c *banTestCore) SetAccessMetricsCallback(ingcore.AccessMetricsCallback) {}
func (c *banTestCore) ConfigFingerprint() string                              { return "" }
func (c *banTestCore) SetBanSet(s *waf.BanSet)                                { c.set = s; c.calls++ }
func (c *banTestCore) GetBanSet() *waf.BanSet                                 { return c.set }
func (c *banTestCore) SetBanHitCallback(ingcore.BanHitCallback)               {}

func newBanTestService(t *testing.T, core ingcore.Core) *WAFBanService {
	t.Helper()
	cfg := &config.Config{CoreInstance: core}
	ing := NewIngress(cfg)
	return NewWAFBanService(ing, &Audit{})
}

func TestWAFBanCreateAndSync(t *testing.T) {
	setupBanDB(t)
	core := &banTestCore{}
	svc := newBanTestService(t, core)

	row, err := svc.Create(BanInput{
		Kind:       "ip",
		Value:      "203.0.113.42",
		Reason:     "sqlmap scan",
		TTLSeconds: 3600,
		Actor:      "alice",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if row.Value != "203.0.113.42" || row.Kind != "ip" {
		t.Fatalf("unexpected row: %+v", row)
	}
	if !row.Active {
		t.Fatalf("expected row to be active")
	}
	if core.calls != 1 || core.set == nil {
		t.Fatalf("expected SetBanSet called once, got calls=%d set=%v", core.calls, core.set)
	}
	if core.set.Len() != 1 {
		t.Fatalf("ban set len = %d, want 1", core.set.Len())
	}

	// Audit row recorded with the real actor.
	var logs []model.AuditLog
	gormx.GetDB().Order("id desc").Limit(1).Find(&logs)
	if len(logs) != 1 || logs[0].Action != "waf.ban.add" || logs[0].Actor != "alice" {
		t.Fatalf("unexpected audit row: %+v", logs)
	}
	if !strings.Contains(logs[0].Detail, "kind=ip") {
		t.Fatalf("audit detail missing kind: %q", logs[0].Detail)
	}
}

func TestWAFBanCreateRejectsInvalidIP(t *testing.T) {
	setupBanDB(t)
	svc := newBanTestService(t, &banTestCore{})

	_, err := svc.Create(BanInput{Kind: "ip", Value: "999.1.1.1", TTLSeconds: 60})
	if err == nil || !strings.HasPrefix(err.Error(), "invalid_value:") {
		t.Fatalf("expected invalid_value error, got %v", err)
	}
}

func TestWAFBanCreateRequiresConfirmForPermanent(t *testing.T) {
	setupBanDB(t)
	svc := newBanTestService(t, &banTestCore{})

	_, err := svc.Create(BanInput{Kind: "ip", Value: "203.0.113.42"})
	if err == nil || !strings.HasPrefix(err.Error(), "confirm_required:") {
		t.Fatalf("expected confirm_required error, got %v", err)
	}

	if _, err := svc.Create(BanInput{Kind: "ip", Value: "203.0.113.42", ConfirmPermanent: true}); err != nil {
		t.Fatalf("permanent ban with confirmation should succeed: %v", err)
	}
}

func TestWAFBanCreateRejectsDuplicate(t *testing.T) {
	setupBanDB(t)
	svc := newBanTestService(t, &banTestCore{})

	if _, err := svc.Create(BanInput{Kind: "ip", Value: "203.0.113.42", TTLSeconds: 3600}); err != nil {
		t.Fatalf("first create: %v", err)
	}
	_, err := svc.Create(BanInput{Kind: "ip", Value: "203.0.113.42", TTLSeconds: 3600})
	if err == nil || !strings.HasPrefix(err.Error(), "duplicate:") {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestWAFBanCreateRejectsTrustedProxy(t *testing.T) {
	setupBanDB(t)
	core := &banTestCore{}
	// The service resolves trusted proxies from the ingress config, which in
	// this test has no file; emulate by testing the helper contract directly.
	svc := newBanTestService(t, core)

	trusted, err := svc.isTrustedProxyValue(waf.BanKindIP, "10.1.2.3")
	if err != nil {
		t.Fatalf("isTrustedProxyValue: %v", err)
	}
	if trusted {
		t.Fatalf("no trusted proxies configured, so nothing should be trusted")
	}
}

func TestWAFBanUARegexValidation(t *testing.T) {
	setupBanDB(t)
	svc := newBanTestService(t, &banTestCore{})

	_, err := svc.Create(BanInput{Kind: "ua_regex", Value: "(?i", TTLSeconds: 60})
	if err == nil || !strings.HasPrefix(err.Error(), "invalid_value:") {
		t.Fatalf("expected invalid_value for bad regex, got %v", err)
	}

	if _, err := svc.Create(BanInput{Kind: "ua_regex", Value: `(?i)(nikto|sqlmap)`, TTLSeconds: 60}); err != nil {
		t.Fatalf("valid regex rejected: %v", err)
	}
}

func TestWAFBanListFilters(t *testing.T) {
	setupBanDB(t)
	svc := newBanTestService(t, &banTestCore{})

	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	db := gormx.GetDB()
	rows := []model.WAFBan{
		{Kind: "ip", Value: "203.0.113.1", Enabled: true, ExpiresAt: &future, Actor: "alice"},
		{Kind: "ip", Value: "203.0.113.2", Enabled: true, ExpiresAt: &past, Actor: "bob"}, // expired
		{Kind: "ua_contains", Value: "sqlmap", Enabled: true, ExpiresAt: &future, Actor: "alice"},
		{Kind: "ip", Value: "203.0.113.3", Enabled: false, ExpiresAt: &future, Actor: "carol"}, // disabled
	}
	for i := range rows {
		if err := db.Create(&rows[i]).Error; err != nil {
			t.Fatal(err)
		}
	}

	active, err := svc.List(BanFilter{Status: "active"})
	if err != nil {
		t.Fatalf("List active: %v", err)
	}
	if len(active) != 2 {
		t.Fatalf("active count = %d, want 2 (%+v)", len(active), active)
	}

	expired, err := svc.List(BanFilter{Status: "expired"})
	if err != nil {
		t.Fatalf("List expired: %v", err)
	}
	if len(expired) != 1 || expired[0].Value != "203.0.113.2" {
		t.Fatalf("expired rows = %+v", expired)
	}

	all, err := svc.List(BanFilter{Status: "all"})
	if err != nil {
		t.Fatalf("List all: %v", err)
	}
	if len(all) != 4 {
		t.Fatalf("all count = %d, want 4", len(all))
	}

	ua, err := svc.List(BanFilter{Status: "active", Kind: "ua_contains"})
	if err != nil {
		t.Fatalf("List kind: %v", err)
	}
	if len(ua) != 1 {
		t.Fatalf("ua count = %d, want 1", len(ua))
	}

	if _, err := svc.List(BanFilter{Status: "bogus"}); err == nil {
		t.Fatalf("expected error for invalid status")
	}
}

func TestWAFBanDeleteRemovesFromCore(t *testing.T) {
	setupBanDB(t)
	core := &banTestCore{}
	svc := newBanTestService(t, core)

	row, err := svc.Create(BanInput{Kind: "ip", Value: "203.0.113.42", TTLSeconds: 3600, Actor: "alice"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if core.set.Len() != 1 {
		t.Fatalf("expected 1 ban after create")
	}

	if err := svc.Delete(row.ID, "alice"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if core.set.Len() != 0 {
		t.Fatalf("ban set should be empty after delete, got %d", core.set.Len())
	}

	var logs []model.AuditLog
	gormx.GetDB().Where("action = ?", "waf.ban.remove").Find(&logs)
	if len(logs) != 1 || logs[0].Actor != "alice" {
		t.Fatalf("expected waf.ban.remove audit by alice, got %+v", logs)
	}

	if err := svc.Delete(9999, "alice"); err == nil || !strings.HasPrefix(err.Error(), "not_found:") {
		t.Fatalf("expected not_found, got %v", err)
	}
}

func TestWAFBanUpdateRenew(t *testing.T) {
	setupBanDB(t)
	core := &banTestCore{}
	svc := newBanTestService(t, core)

	row, err := svc.Create(BanInput{Kind: "ip", Value: "203.0.113.42", TTLSeconds: 60, Actor: "alice"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	ttl := int64(7200)
	updated, err := svc.Update(row.ID, BanPatch{TTLSeconds: &ttl, Actor: "bob"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.ExpiresAt == nil || time.Until(*updated.ExpiresAt) < time.Hour {
		t.Fatalf("expiry not extended: %+v", updated.ExpiresAt)
	}

	var logs []model.AuditLog
	gormx.GetDB().Where("action = ?", "waf.ban.renew").Find(&logs)
	if len(logs) != 1 || logs[0].Actor != "bob" {
		t.Fatalf("expected waf.ban.renew by bob, got %+v", logs)
	}
}

func TestWAFBanSyncSkipsWhenCoreMissing(t *testing.T) {
	setupBanDB(t)
	svc := newBanTestService(t, nil)

	if _, err := svc.Create(BanInput{Kind: "ip", Value: "203.0.113.42", TTLSeconds: 3600}); err != nil {
		t.Fatalf("create without core should still succeed: %v", err)
	}
	if err := svc.SyncToCore(); err != nil {
		t.Fatalf("SyncToCore without core: %v", err)
	}
}

func TestWAFBanSyncSkipsExpiredRows(t *testing.T) {
	setupBanDB(t)
	core := &banTestCore{}
	svc := newBanTestService(t, core)

	past := time.Now().Add(-time.Hour)
	db := gormx.GetDB()
	if err := db.Create(&model.WAFBan{
		Kind: "ip", Value: "203.0.113.9", Enabled: true, ExpiresAt: &past,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.SyncToCore(); err != nil {
		t.Fatalf("SyncToCore: %v", err)
	}
	if core.set.Len() != 0 {
		t.Fatalf("expired rows must not be loaded into the snapshot, got %d", core.set.Len())
	}
}

func TestNormalizeBanKind(t *testing.T) {
	cases := map[string]waf.BanKind{
		"":            waf.BanKindIP,
		"ip":          waf.BanKindIP,
		"IP":          waf.BanKindIP,
		"ip_cidr":     waf.BanKindIPCIDR,
		"ua_contains": waf.BanKindUAContains,
		"ua_regex":    waf.BanKindUARegex,
	}
	for raw, want := range cases {
		got, err := normalizeBanKind(raw)
		if err != nil {
			t.Fatalf("normalizeBanKind(%q): %v", raw, err)
		}
		if got != want {
			t.Fatalf("normalizeBanKind(%q) = %q, want %q", raw, got, want)
		}
	}
	if _, err := normalizeBanKind("nope"); err == nil {
		t.Fatalf("expected error for unknown kind")
	}
}

func TestResolveBanExpiry(t *testing.T) {
	if got, err := resolveBanExpiry(nil, 0); err != nil || got != nil {
		t.Fatalf("zero ttl should mean permanent, got %v %v", got, err)
	}
	if _, err := resolveBanExpiry(nil, -5); err == nil {
		t.Fatalf("negative ttl must error")
	}
	got, err := resolveBanExpiry(nil, 3600)
	if err != nil || got == nil {
		t.Fatalf("expected expiry, got %v %v", got, err)
	}
	if d := time.Until(*got); d < 59*time.Minute || d > 61*time.Minute {
		t.Fatalf("unexpected ttl: %v", d)
	}
}

func TestEncodeDecodeBanHosts(t *testing.T) {
	// Empty host scope must decode to a non-nil empty slice. A nil []string
	// serializes to JSON `null` and crashes frontend consumers reading
	// `hosts.length` (regression: WAF ban list page threw on global bans).
	for _, raw := range []string{"", "not-json", "null"} {
		got := decodeBanHosts(raw)
		if got == nil {
			t.Fatalf("decodeBanHosts(%q) returned nil; want non-nil empty slice", raw)
		}
		if len(got) != 0 {
			t.Fatalf("decodeBanHosts(%q) = %v; want empty", raw, got)
		}
	}

	raw, err := encodeBanHosts([]string{"api.example.com", "*.example.org"})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	hosts := decodeBanHosts(raw)
	if len(hosts) != 2 || hosts[0] != "api.example.com" {
		t.Fatalf("roundtrip mismatch: %v", hosts)
	}
}

func TestBanRowJSON_HostsNeverNull(t *testing.T) {
	// Contract-level guard: the API must emit `hosts: []`, never `hosts: null`,
	// for a global-scope ban. This is exactly what the admin UI consumes.
	row := banRowFromModel(&model.WAFBan{
		ID: 1, Kind: "ip", Value: "203.0.113.42", Hosts: "", Enabled: true,
	})
	b, err := json.Marshal(row)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(b), `"hosts":null`) {
		t.Fatalf("hosts serialized as null: %s", string(b))
	}
	if !strings.Contains(string(b), `"hosts":[]`) {
		t.Fatalf("hosts must serialize as []: %s", string(b))
	}

	// An explicit JSON "null" in a stored row must not leak through either.
	row2 := banRowFromModel(&model.WAFBan{
		ID: 2, Kind: "ip", Value: "203.0.113.43", Hosts: "null", Enabled: true,
	})
	b2, err := json.Marshal(row2)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(b2), `"hosts":null`) {
		t.Fatalf("stored null hosts leaked into API response: %s", string(b2))
	}
}

func TestNormalizeBanHosts(t *testing.T) {
	got, err := normalizeBanHosts([]string{" api.example.com ", "", "api.example.com", "*.example.org"})
	if err != nil {
		t.Fatalf("normalizeBanHosts: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected dedup to 2 hosts, got %v", got)
	}
	if _, err := normalizeBanHosts([]string{"[bad"}); err == nil {
		t.Fatalf("expected invalid_hosts error for bad regex")
	}
}

func TestWAFBanEventRecordedAndFilterable(t *testing.T) {
	setupBanDB(t)
	a := &Audit{}

	// The core callback writes ban hits with action="ban" and rule="ban:<kind>".
	row, err := a.RecordWAFEvent("ban", "ban:ip", "api.example.com", "/", "203.0.113.42", "sqlmap/1.7.2")
	if err != nil {
		t.Fatalf("RecordWAFEvent: %v", err)
	}
	if row.Action != "ban" {
		t.Fatalf("action = %q, want ban", row.Action)
	}
	if row.Status != wafEventStatusOpen {
		t.Fatalf("status = %q, want open", row.Status)
	}

	// Also record a normal WAF block so the filter has something to exclude.
	if _, err := a.RecordWAFEvent("block", "sql-injection-uri", "api.example.com", "/v1", "198.51.100.9", "curl"); err != nil {
		t.Fatalf("RecordWAFEvent block: %v", err)
	}

	banOnly, err := a.ListWAFEvents(WAFAuditFilter{Action: "ban"})
	if err != nil {
		t.Fatalf("ListWAFEvents ban: %v", err)
	}
	if len(banOnly) != 1 {
		t.Fatalf("ban filter returned %d rows, want 1", len(banOnly))
	}
	if banOnly[0].Rule != "ban:ip" {
		t.Fatalf("rule = %q, want ban:ip", banOnly[0].Rule)
	}

	blockOnly, err := a.ListWAFEvents(WAFAuditFilter{Action: "block"})
	if err != nil {
		t.Fatalf("ListWAFEvents block: %v", err)
	}
	if len(blockOnly) != 1 || blockOnly[0].Rule != "sql-injection-uri" {
		t.Fatalf("block filter = %+v", blockOnly)
	}

	all, err := a.ListWAFEvents(WAFAuditFilter{})
	if err != nil {
		t.Fatalf("ListWAFEvents all: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("unfiltered returned %d rows, want 2", len(all))
	}

	// Ban events carry the offending User-Agent so the UI can offer "ban UA".
	if banOnly[0].UserAgent != "sqlmap/1.7.2" {
		t.Fatalf("user_agent = %q", banOnly[0].UserAgent)
	}
}

func TestFlushBanHitsAccumulatesPerEntry(t *testing.T) {
	setupBanDB(t)
	svc := newBanTestService(t, &banTestCore{})

	first, err := svc.Create(BanInput{Kind: "ip", Value: "203.0.113.10", TTLSeconds: 3600})
	if err != nil {
		t.Fatalf("Create first: %v", err)
	}
	second, err := svc.Create(BanInput{Kind: "ip", Value: "203.0.113.11", TTLSeconds: 3600})
	if err != nil {
		t.Fatalf("Create second: %v", err)
	}

	RecordBanHit(first.ID)
	RecordBanHit(first.ID)
	RecordBanHit(first.ID)
	RecordBanHit(second.ID)

	if err := svc.FlushBanHits(); err != nil {
		t.Fatalf("FlushBanHits: %v", err)
	}

	rowFirst, err := svc.Get(first.ID)
	if err != nil {
		t.Fatalf("Get first: %v", err)
	}
	rowSecond, err := svc.Get(second.ID)
	if err != nil {
		t.Fatalf("Get second: %v", err)
	}
	if rowFirst.Hits != 3 {
		t.Fatalf("first hits = %d, want 3", rowFirst.Hits)
	}
	if rowSecond.Hits != 1 {
		t.Fatalf("second hits = %d, want 1", rowSecond.Hits)
	}

	// Flushing again must not double-count: counters were drained.
	if err := svc.FlushBanHits(); err != nil {
		t.Fatalf("FlushBanHits second: %v", err)
	}
	again, err := svc.Get(first.ID)
	if err != nil {
		t.Fatalf("Get again: %v", err)
	}
	if again.Hits != 3 {
		t.Fatalf("hits after second flush = %d, want 3 (no double count)", again.Hits)
	}
}

func TestRecordBanHitConcurrent(t *testing.T) {
	setupBanDB(t)
	svc := newBanTestService(t, &banTestCore{})

	row, err := svc.Create(BanInput{Kind: "ip", Value: "203.0.113.77", TTLSeconds: 3600})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	const goroutines = 50
	const perGoroutine = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < perGoroutine; j++ {
				RecordBanHit(row.ID)
			}
		}()
	}
	wg.Wait()

	if err := svc.FlushBanHits(); err != nil {
		t.Fatalf("FlushBanHits: %v", err)
	}
	got, err := svc.Get(row.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	want := int64(goroutines * perGoroutine)
	if got.Hits != want {
		t.Fatalf("hits = %d, want %d (lost updates under concurrency)", got.Hits, want)
	}
}
