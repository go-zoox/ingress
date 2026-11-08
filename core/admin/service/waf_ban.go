package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-zoox/gormx"
	ingcore "github.com/go-zoox/ingress/core"
	"github.com/go-zoox/ingress/core/admin/model"
	"github.com/go-zoox/ingress/core/waf"
	"gorm.io/gorm"
)

// Ban statuses accepted by the list filter.
const (
	BanStatusActive  = "active"
	BanStatusExpired = "expired"
	BanStatusAll     = "all"
)

// BanInput is the create payload for a runtime ban.
type BanInput struct {
	Kind   string   `json:"kind"`
	Value  string   `json:"value"`
	Hosts  []string `json:"hosts"`
	Reason string   `json:"reason"`
	// ExpiresAt is the absolute expiry; nil means permanent and requires
	// ConfirmPermanent.
	ExpiresAt *time.Time `json:"expires_at"`
	// TTLSeconds is a convenience alternative to ExpiresAt.
	TTLSeconds int64 `json:"ttl_seconds"`
	// ConfirmPermanent must be true when creating a permanent ban.
	ConfirmPermanent bool `json:"confirm_permanent"`
	Actor            string
}

// BanPatch is the update payload (TTL / enable switch).
type BanPatch struct {
	ExpiresAt        *time.Time `json:"expires_at"`
	TTLSeconds       *int64     `json:"ttl_seconds"`
	Enabled          *bool      `json:"enabled"`
	ConfirmPermanent bool       `json:"confirm_permanent"`
	Actor            string
}

// BanFilter narrows the list query.
type BanFilter struct {
	Status  string // active | expired | all
	Kind    string
	Keyword string
	Limit   int
}

// BanRow is a list row for the admin UI.
type BanRow struct {
	ID        uint       `json:"id"`
	Kind      string     `json:"kind"`
	Value     string     `json:"value"`
	Hosts     []string   `json:"hosts"`
	Scope     string     `json:"scope"` // global | hosts
	Reason    string     `json:"reason"`
	Source    string     `json:"source"`
	Actor     string     `json:"actor"`
	Hits      int64      `json:"hits"`
	Enabled   bool       `json:"enabled"`
	ExpiresAt *time.Time `json:"expires_at"`
	Active    bool       `json:"active"`
	CreatedAt time.Time  `json:"created_at"`
}

// BanPreview estimates the impact of a prospective ban.
type BanPreview struct {
	Kind          string   `json:"kind"`
	Value         string   `json:"value"`
	Normalized    string   `json:"normalized"`
	Hits24h       int64    `json:"hits_24h"`
	Hosts         int64    `json:"hosts"`
	DistinctUA    int64    `json:"distinct_ua"`
	Paths         int64    `json:"paths"`
	TrustedProxy  bool     `json:"trusted_proxy"`
	InAllowList   bool     `json:"in_allow_list"`
	Warnings      []string `json:"warnings"`
	Blocked       bool     `json:"blocked"`
	BlockedReason string   `json:"blocked_reason,omitempty"`
}

// WAFBanService manages the runtime ban list and keeps the running core in sync.
type WAFBanService struct {
	ingress *Ingress
	audit   *Audit
}

// NewWAFBanService builds the service.
func NewWAFBanService(ingress *Ingress, audit *Audit) *WAFBanService {
	return &WAFBanService{ingress: ingress, audit: audit}
}

// bannedClientIPs counts ban hits per entry inside the process so the request
// path never writes to the database.
var bannedClientIPs = struct {
	sync.Mutex
	counts map[uint]int64
}{counts: map[uint]int64{}}

// RecordBanHit increments the in-process hit counter for one ban entry.
func RecordBanHit(id uint) {
	if id == 0 {
		return
	}
	bannedClientIPs.Lock()
	bannedClientIPs.counts[id]++
	bannedClientIPs.Unlock()
}

// trustProxiesForBans resolves the trusted proxy CIDRs used for ban client IP
// detection. Malformed values are ignored here: config validation already
// rejects them, and a bad value must not break ban syncing.
func (s *WAFBanService) trustProxiesForBans() []*net.IPNet {
	if s == nil || s.ingress == nil {
		return nil
	}
	cfg, err := s.ingress.LoadConfig()
	if err != nil || cfg == nil {
		return nil
	}
	nets, err := waf.ParseTrustedProxyNets(cfg.WAF.TrustedProxies)
	if err != nil {
		return nil
	}
	return nets
}

// SyncToCore recompiles the active ban list and installs it on the running core.
//
// The swap is atomic and does not reload the config, flush caches, or reset rate
// limiters. It returns an error only for compile failures, which mean the DB
// holds an entry the WAF package cannot compile.
func (s *WAFBanService) SyncToCore() error {
	core := s.coreInstance()
	if core == nil {
		return nil // admin running standalone: nothing to inject into
	}

	// Enabled rows are loaded without a SQL time predicate and filtered in Go.
	// SQLite stores expires_at as a local-time string, so comparing it against a
	// UTC time.Time in SQL is off by the zone offset near the boundary.
	var rows []model.WAFBan
	if err := gormx.GetDB().Where("enabled = ?", true).Find(&rows).Error; err != nil {
		return err
	}

	now := time.Now()
	entries := make([]waf.BanEntry, 0, len(rows))
	for _, row := range rows {
		if row.ExpiresAt != nil && !row.ExpiresAt.After(now) {
			continue
		}
		entries = append(entries, waf.BanEntry{
			ID:        row.ID,
			Kind:      waf.BanKind(row.Kind),
			Value:     row.Value,
			Hosts:     decodeBanHosts(row.Hosts),
			ExpiresAt: derefTimePtr(row.ExpiresAt),
			Reason:    row.Reason,
		})
	}

	set, err := waf.CompileBans(entries, s.trustProxiesForBans())
	if err != nil {
		return fmt.Errorf("compile bans: %w", err)
	}
	core.SetBanSet(set)
	return nil
}

func (s *WAFBanService) coreInstance() ingcore.Core {
	if s == nil || s.ingress == nil || s.ingress.cfg == nil {
		return nil
	}
	return s.ingress.cfg.CoreInstance
}

// Create validates and stores a ban, then syncs the running core synchronously
// so a 200 response guarantees the ban is already enforced.
func (s *WAFBanService) Create(in BanInput) (*BanRow, error) {
	kind, err := normalizeBanKind(in.Kind)
	if err != nil {
		return nil, err
	}
	value := strings.TrimSpace(in.Value)
	if err := waf.ValidateBanValue(kind, value); err != nil {
		return nil, fmt.Errorf("invalid_value: %w", err)
	}

	hosts, err := normalizeBanHosts(in.Hosts)
	if err != nil {
		return nil, err
	}

	expiresAt, err := resolveBanExpiry(in.ExpiresAt, in.TTLSeconds)
	if err != nil {
		return nil, err
	}
	if expiresAt == nil && !in.ConfirmPermanent {
		return nil, fmt.Errorf("confirm_required: permanent bans require confirm_permanent")
	}

	if trusted, err := s.isTrustedProxyValue(kind, value); err != nil {
		return nil, err
	} else if trusted {
		return nil, fmt.Errorf("trusted_proxy: %s is a trusted proxy (waf.trusted_proxies); banning it would block the whole site", value)
	}

	hostsJSON, err := encodeBanHosts(hosts)
	if err != nil {
		return nil, err
	}

	exists, err := s.duplicateExists(kind, value, hostsJSON)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("duplicate: an active ban for %s %q already exists", kind, value)
	}

	row := &model.WAFBan{
		Kind:      string(kind),
		Value:     value,
		Hosts:     hostsJSON,
		Reason:    strings.TrimSpace(in.Reason),
		Source:    waf.BanSourceManual,
		Actor:     in.Actor,
		Enabled:   true,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}
	if err := gormx.GetDB().Create(row).Error; err != nil {
		return nil, err
	}

	if err := s.SyncToCore(); err != nil {
		return nil, err
	}

	_ = s.audit.Record("waf.ban.add", banAuditDetail(row, expiresAt), in.Actor)
	out := banRowFromModel(row)
	return &out, nil
}

// Update changes a ban's expiry or enabled flag.
func (s *WAFBanService) Update(id uint, in BanPatch) (*BanRow, error) {
	var row model.WAFBan
	if err := gormx.GetDB().First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("not_found: ban %d", id)
		}
		return nil, err
	}

	updates := map[string]any{"updated_at": time.Now()}
	action := ""

	if in.ExpiresAt != nil || in.TTLSeconds != nil {
		expiresAt, err := resolveBanExpiry(in.ExpiresAt, int64OrZero(in.TTLSeconds))
		if err != nil {
			return nil, err
		}
		if expiresAt == nil && !in.ConfirmPermanent {
			return nil, fmt.Errorf("confirm_required: permanent bans require confirm_permanent")
		}
		updates["expires_at"] = expiresAt
		row.ExpiresAt = expiresAt
		action = "waf.ban.renew"
	}

	if in.Enabled != nil {
		updates["enabled"] = *in.Enabled
		row.Enabled = *in.Enabled
		if action == "" {
			if *in.Enabled {
				action = "waf.ban.add"
			} else {
				action = "waf.ban.remove"
			}
		}
	}

	if err := gormx.GetDB().Model(&model.WAFBan{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.SyncToCore(); err != nil {
		return nil, err
	}

	if action != "" {
		_ = s.audit.Record(action, banAuditDetail(&row, row.ExpiresAt), in.Actor)
	}
	out := banRowFromModel(&row)
	return &out, nil
}

// Delete removes a ban and restores access immediately.
func (s *WAFBanService) Delete(id uint, actor string) error {
	var row model.WAFBan
	if err := gormx.GetDB().First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("not_found: ban %d", id)
		}
		return err
	}
	if err := gormx.GetDB().Delete(&model.WAFBan{}, id).Error; err != nil {
		return err
	}
	if err := s.SyncToCore(); err != nil {
		return err
	}
	_ = s.audit.Record("waf.ban.remove", fmt.Sprintf("解封 ban#%d kind=%s value=%s", row.ID, row.Kind, row.Value), actor)
	return nil
}

// List returns ban rows filtered by status/kind/keyword.
func (s *WAFBanService) List(f BanFilter) ([]BanRow, error) {
	q := gormx.GetDB().Model(&model.WAFBan{})

	status := strings.TrimSpace(strings.ToLower(f.Status))
	now := time.Now()
	switch status {
	case "", BanStatusActive:
		// Only the enabled flag is pushed into SQL; TTL is filtered in Go
		// because SQLite stores expires_at as a local-time string.
		q = q.Where("enabled = ?", true)
	case BanStatusExpired:
		// no SQL filter on time; applied below
	case BanStatusAll:
		// no filter
	default:
		return nil, fmt.Errorf("invalid status %q (use active, expired, or all)", f.Status)
	}

	if kind := strings.TrimSpace(strings.ToLower(f.Kind)); kind != "" && kind != "all" {
		q = q.Where("kind = ?", kind)
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("value LIKE ? OR reason LIKE ? OR actor LIKE ?", like, like, like)
	}

	limit := f.Limit
	if limit <= 0 || limit > 500 {
		limit = 200
	}

	var rows []model.WAFBan
	if err := q.Order("created_at desc, id desc").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]BanRow, 0, len(rows))
	for i := range rows {
		row := &rows[i]
		expired := row.ExpiresAt != nil && !row.ExpiresAt.After(now)
		switch status {
		case BanStatusActive:
			if expired {
				continue
			}
		case BanStatusExpired:
			if !expired {
				continue
			}
		}
		out = append(out, banRowFromModel(row))
	}
	return out, nil
}

// Get returns one ban row.
func (s *WAFBanService) Get(id uint) (*BanRow, error) {
	var row model.WAFBan
	if err := gormx.GetDB().First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("not_found: ban %d", id)
		}
		return nil, err
	}
	out := banRowFromModel(&row)
	return &out, nil
}

// Preview estimates impact for a prospective ban before it is created.
func (s *WAFBanService) Preview(in BanInput) (BanPreview, error) {
	kind, err := normalizeBanKind(in.Kind)
	if err != nil {
		return BanPreview{}, err
	}
	value := strings.TrimSpace(in.Value)
	if err := waf.ValidateBanValue(kind, value); err != nil {
		return BanPreview{}, fmt.Errorf("invalid_value: %w", err)
	}

	preview := BanPreview{Kind: string(kind), Value: value, Normalized: value}
	if kind == waf.BanKindIP {
		if ip := net.ParseIP(value); ip != nil {
			if v4 := ip.To4(); v4 != nil {
				preview.Normalized = v4.String()
			}
		}
	}

	// Impact: how often did this target show up in the last 24h?
	since := time.Now().Add(-24 * time.Hour)
	q := gormx.GetDB().Model(&model.WAFEvent{}).Where("created_at >= ?", since)
	switch kind {
	case waf.BanKindIP, waf.BanKindIPCIDR:
		q = q.Where("client_ip = ?", preview.Normalized)
	case waf.BanKindUAContains:
		q = q.Where("user_agent LIKE ?", "%"+value+"%")
	case waf.BanKindUARegex:
		// Regex cannot be pushed into SQL portably; approximate with no filter
		// and let the caller rely on the warning copy instead.
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return BanPreview{}, err
	}
	preview.Hits24h = total

	if total > 0 {
		var hostCount int64
		if err := q.Distinct("host").Count(&hostCount).Error; err == nil {
			preview.Hosts = hostCount
		}
		var uaCount int64
		if err := q.Distinct("user_agent").Count(&uaCount).Error; err == nil {
			preview.DistinctUA = uaCount
		}
		var pathCount int64
		if err := q.Distinct("path").Count(&pathCount).Error; err == nil {
			preview.Paths = pathCount
		}
	}

	trusted, err := s.isTrustedProxyValue(kind, value)
	if err != nil {
		return BanPreview{}, err
	}
	preview.TrustedProxy = trusted
	if trusted {
		preview.Blocked = true
		preview.BlockedReason = "该地址是反向代理/负载均衡自身（waf.trusted_proxies），封禁会导致整站不可访问"
		preview.Warnings = append(preview.Warnings, preview.BlockedReason)
	}

	// NAT / carrier-grade NAT heuristic: one IP, many distinct UAs.
	if (kind == waf.BanKindIP || kind == waf.BanKindIPCIDR) && preview.DistinctUA >= 20 {
		preview.Warnings = append(preview.Warnings,
			fmt.Sprintf("该 IP 近 24h 出现 %d 个不同 UA，可能位于 NAT / 运营商出口之后，封禁可能误伤大量真实用户", preview.DistinctUA))
	}

	if kind == waf.BanKindUARegex || kind == waf.BanKindUAContains {
		preview.Warnings = append(preview.Warnings,
			"UA 是客户端可控字段，更换 UA 即可绕过，按 UA 封禁仅作止血手段")
	}

	if inAllow, err := s.inAllowList(kind, value); err == nil && inAllow {
		preview.InAllowList = true
		preview.Warnings = append(preview.Warnings,
			"该目标在 allow 白名单中；运行时封禁优先于 allow，依然会生效")
	}

	return preview, nil
}

// isTrustedProxyValue reports whether value falls inside waf.trusted_proxies.
func (s *WAFBanService) isTrustedProxyValue(kind waf.BanKind, value string) (bool, error) {
	if kind != waf.BanKindIP && kind != waf.BanKindIPCIDR {
		return false, nil
	}
	nets := s.trustProxiesForBans()
	if len(nets) == 0 {
		return false, nil
	}

	var ip net.IP
	if kind == waf.BanKindIP {
		if v4 := net.ParseIP(value).To4(); v4 != nil {
			ip = v4
		} else {
			ip = net.ParseIP(value)
		}
	} else {
		_, ipNet, err := net.ParseCIDR(value)
		if err != nil {
			return false, nil
		}
		ip = ipNet.IP
	}
	if ip == nil {
		return false, nil
	}
	for _, n := range nets {
		if n.Contains(ip) {
			return true, nil
		}
	}
	return false, nil
}

// inAllowList reports whether the value is covered by the global WAF allow list.
func (s *WAFBanService) inAllowList(kind waf.BanKind, value string) (bool, error) {
	if kind != waf.BanKindIP && kind != waf.BanKindIPCIDR {
		return false, nil
	}
	cfg, err := s.ingress.LoadConfig()
	if err != nil || cfg == nil {
		return false, nil
	}
	ip := net.ParseIP(value)
	if kind == waf.BanKindIPCIDR {
		_, ipNet, err := net.ParseCIDR(value)
		if err != nil {
			return false, nil
		}
		ip = ipNet.IP
	}
	if ip == nil {
		return false, nil
	}
	for _, raw := range cfg.WAF.Allow {
		if _, n, err := net.ParseCIDR(raw); err == nil {
			if n.Contains(ip) {
				return true, nil
			}
			continue
		}
		if allowed := net.ParseIP(raw); allowed != nil && allowed.Equal(ip) {
			return true, nil
		}
	}
	return false, nil
}

// duplicateExists reports whether an active ban with the same identity exists.
func (s *WAFBanService) duplicateExists(kind waf.BanKind, value, hostsJSON string) (bool, error) {
	now := time.Now()
	// Time filtering is done in Go for the same SQLite timezone reason as above.
	var rows []model.WAFBan
	if err := gormx.GetDB().Model(&model.WAFBan{}).
		Where("kind = ?", string(kind)).
		Where("value = ?", value).
		Where("hosts = ?", hostsJSON).
		Where("enabled = ?", true).
		Find(&rows).Error; err != nil {
		return false, err
	}
	for i := range rows {
		if rows[i].ExpiresAt == nil || rows[i].ExpiresAt.After(now) {
			return true, nil
		}
	}
	return false, nil
}

// FlushBanHits persists in-process hit counters into the database.
func (s *WAFBanService) FlushBanHits() error {
	bannedClientIPs.Lock()
	if len(bannedClientIPs.counts) == 0 {
		bannedClientIPs.Unlock()
		return nil
	}
	snapshot := bannedClientIPs.counts
	bannedClientIPs.counts = map[uint]int64{}
	bannedClientIPs.Unlock()

	ids := make([]uint, 0, len(snapshot))
	for id := range snapshot {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	for _, id := range ids {
		n := snapshot[id]
		if n <= 0 {
			continue
		}
		if err := gormx.GetDB().Model(&model.WAFBan{}).
			Where("id = ?", id).
			UpdateColumn("hits", gorm.Expr("hits + ?", n)).Error; err != nil {
			return err
		}
	}
	return nil
}

// ---- helpers ----

func normalizeBanKind(raw string) (waf.BanKind, error) {
	kind := waf.BanKind(strings.TrimSpace(strings.ToLower(raw)))
	switch kind {
	case "":
		return waf.BanKindIP, nil
	case waf.BanKindIP, waf.BanKindIPCIDR, waf.BanKindUAContains, waf.BanKindUARegex:
		return kind, nil
	default:
		return "", fmt.Errorf("invalid_kind: %q (use ip, ip_cidr, ua_contains, or ua_regex)", raw)
	}
}

func normalizeBanHosts(hosts []string) ([]string, error) {
	out := make([]string, 0, len(hosts))
	seen := map[string]struct{}{}
	for _, h := range hosts {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}
		if _, ok := seen[h]; ok {
			continue
		}
		seen[h] = struct{}{}
		out = append(out, h)
	}
	if len(out) == 0 {
		return nil, nil
	}
	// Reject malformed host patterns early by compiling them once.
	if _, err := waf.CompileBanHosts(out); err != nil {
		return nil, fmt.Errorf("invalid_hosts: %w", err)
	}
	return out, nil
}

func resolveBanExpiry(explicit *time.Time, ttlSeconds int64) (*time.Time, error) {
	if explicit != nil {
		t := explicit.UTC()
		return &t, nil
	}
	if ttlSeconds > 0 {
		t := time.Now().Add(time.Duration(ttlSeconds) * time.Second).UTC()
		return &t, nil
	}
	if ttlSeconds < 0 {
		return nil, fmt.Errorf("invalid_ttl: ttl_seconds must be positive")
	}
	return nil, nil // permanent
}

func int64OrZero(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

func encodeBanHosts(hosts []string) (string, error) {
	if len(hosts) == 0 {
		return "", nil
	}
	b, err := json.Marshal(hosts)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func decodeBanHosts(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

func derefTimePtr(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

func banRowFromModel(row *model.WAFBan) BanRow {
	hosts := decodeBanHosts(row.Hosts)
	active := row.Enabled && (row.ExpiresAt == nil || row.ExpiresAt.After(time.Now()))
	scope := "global"
	if len(hosts) > 0 {
		scope = "hosts"
	}
	return BanRow{
		ID:        row.ID,
		Kind:      row.Kind,
		Value:     row.Value,
		Hosts:     hosts,
		Scope:     scope,
		Reason:    row.Reason,
		Source:    row.Source,
		Actor:     row.Actor,
		Hits:      row.Hits,
		Enabled:   row.Enabled,
		ExpiresAt: row.ExpiresAt,
		Active:    active,
		CreatedAt: row.CreatedAt,
	}
}

func banAuditDetail(row *model.WAFBan, expiresAt *time.Time) string {
	ttl := "permanent"
	if expiresAt != nil {
		ttl = expiresAt.Format(time.RFC3339)
	}
	scope := "global"
	if hosts := decodeBanHosts(row.Hosts); len(hosts) > 0 {
		scope = strings.Join(hosts, ",")
	}
	return fmt.Sprintf("kind=%s value=%s scope=%s expires=%s reason=%s", row.Kind, row.Value, scope, ttl, row.Reason)
}
