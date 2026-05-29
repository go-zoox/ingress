package service

import (
	"strings"
	"time"

	"github.com/go-zoox/gormx"
	"github.com/go-zoox/ingress/core/admin/model"
	"gorm.io/gorm"
)

const (
	wafEventStatusOpen     = "open"
	wafEventStatusIgnored  = "ignored"
	wafEventStatusResolved = "resolved"
)

// WAFAuditFilter holds optional filters for listing WAF events.
type WAFAuditFilter struct {
	Action    string
	Host      string
	ClientIP  string
	Rule      string
	Path      string // added for frontend path filter
	Status    string // open | ignored | resolved
	TimeStart string // RFC3339 or "2006-01-02"
	TimeEnd   string
	Limit     int
}

// AuditLogRow is a list entry for the admin change timeline.
type AuditLogRow struct {
	ID        uint      `json:"id"`
	Action    string    `json:"action"`
	Detail    string    `json:"detail"`
	Actor     string    `json:"actor"`
	CreatedAt time.Time `json:"created_at"`
}

// Audit writes admin audit rows.
type Audit struct{}

func NewAudit() *Audit {
	return &Audit{}
}

func (a *Audit) List(limit int) ([]AuditLogRow, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows []model.AuditLog
	if err := gormx.GetDB().Order("created_at desc, id desc").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]AuditLogRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, AuditLogRow{
			ID:        row.ID,
			Action:    row.Action,
			Detail:    row.Detail,
			Actor:     row.Actor,
			CreatedAt: row.CreatedAt,
		})
	}
	return out, nil
}

func (a *Audit) Record(action, detail, actor string) error {
	row := &model.AuditLog{
		Action:    action,
		Detail:    detail,
		Actor:     actor,
		CreatedAt: time.Now(),
	}
	return gormx.GetDB().Create(row).Error
}

func (a *Audit) RecordWAFEvent(action, rule, host, path, clientIP, userAgent string) (*model.WAFEvent, error) {
	row := &model.WAFEvent{
		Action:    action,
		Rule:      rule,
		Host:      host,
		Path:      path,
		ClientIP:  clientIP,
		UserAgent: userAgent,
		Status:    wafEventStatusOpen,
		CreatedAt: time.Now(),
	}
	if err := gormx.GetDB().Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

// ListWAFEvents queries WAF events with optional filters.
// Supported filters: action, host, client_ip, rule, time_start, time_end, limit.
func (a *Audit) ListWAFEvents(f WAFAuditFilter) ([]model.WAFEvent, error) {
	if f.Limit <= 0 {
		f.Limit = 50
	}
	var rows []model.WAFEvent
	q := gormx.GetDB().Model(&model.WAFEvent{})
	if f.Action != "" {
		q = q.Where("action = ?", f.Action)
	}
	if f.Host != "" {
		q = q.Where("host LIKE ?", "%"+f.Host+"%")
	}
	if f.ClientIP != "" {
		q = q.Where("client_ip LIKE ?", "%"+f.ClientIP+"%")
	}
	if f.Rule != "" {
		q = q.Where("rule LIKE ?", "%"+f.Rule+"%")
	}
	if f.Path != "" {
		q = q.Where("path LIKE ?", "%"+f.Path+"%")
	}
	if f.Status != "" {
		q = applyWAFEventStatusFilter(q, f.Status)
	}
	if f.TimeStart != "" {
		q = q.Where("created_at >= ?", f.TimeStart)
	}
	if f.TimeEnd != "" {
		q = q.Where("created_at <= ?", f.TimeEnd)
	}
	err := q.Order("created_at desc").Limit(f.Limit).Find(&rows).Error
	return rows, err
}

// GetWAFEvent returns one WAF event by id.
func (a *Audit) GetWAFEvent(id uint) (*model.WAFEvent, error) {
	var row model.WAFEvent
	err := gormx.GetDB().First(&row, id).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// SetWAFEventStatus marks a WAF event as ignored, resolved, or reopened (open).
func (a *Audit) SetWAFEventStatus(id uint, status, note string) (*model.WAFEvent, error) {
	status = strings.TrimSpace(status)
	switch status {
	case wafEventStatusIgnored, wafEventStatusResolved, wafEventStatusOpen:
	default:
		status = wafEventStatusIgnored
	}
	var row model.WAFEvent
	if err := gormx.GetDB().First(&row, id).Error; err != nil {
		return nil, err
	}
	now := time.Now()
	updates := map[string]any{
		"status": status,
		"note":   strings.TrimSpace(note),
	}
	if status == wafEventStatusResolved {
		updates["resolved_at"] = now
	} else {
		updates["resolved_at"] = nil
	}
	if err := gormx.GetDB().Model(&row).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := gormx.GetDB().First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// BatchSetWAFEventStatus updates many WAF events to the same status and optional note.
func (a *Audit) BatchSetWAFEventStatus(ids []uint, status, note string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	status = strings.TrimSpace(status)
	switch status {
	case wafEventStatusIgnored, wafEventStatusResolved, wafEventStatusOpen:
	default:
		status = wafEventStatusIgnored
	}
	now := time.Now()
	updates := map[string]any{
		"status": status,
		"note":   strings.TrimSpace(note),
	}
	if status == wafEventStatusResolved {
		updates["resolved_at"] = now
	} else {
		updates["resolved_at"] = nil
	}
	res := gormx.GetDB().Model(&model.WAFEvent{}).Where("id IN ?", ids).Updates(updates)
	return res.RowsAffected, res.Error
}

// OpenBlockCount returns block events still needing attention.
func (a *Audit) OpenBlockCount() (int64, error) {
	var count int64
	err := gormx.GetDB().Model(&model.WAFEvent{}).
		Where("action = ?", "block").
		Where("(status = ? OR status = '' OR status IS NULL)", wafEventStatusOpen).
		Count(&count).Error
	return count, err
}

func applyWAFEventStatusFilter(q *gorm.DB, status string) *gorm.DB {
	status = strings.TrimSpace(status)
	if status == wafEventStatusOpen {
		return q.Where("(status = ? OR status = '' OR status IS NULL)", wafEventStatusOpen)
	}
	return q.Where("status = ?", status)
}

// distinctWAFHosts returns distinct host values from waf_events for filter dropdowns.
func (a *Audit) DistinctWAFHosts() ([]string, error) {
	var hosts []string
	err := gormx.GetDB().Model(&model.WAFEvent{}).
		Distinct("host").Where("host <> ?", "").
		Order("host asc").Pluck("host", &hosts).Error
	return hosts, err
}

// distinctWAFRules returns distinct rule values from waf_events.
func (a *Audit) DistinctWAFRules() ([]string, error) {
	var rules []string
	err := gormx.GetDB().Model(&model.WAFEvent{}).
		Distinct("rule").Where("rule <> ?", "").
		Order("rule asc").Pluck("rule", &rules).Error
	return rules, err
}

// WAFEventsSummary returns action counts for the dashboard cards.
func (a *Audit) WAFEventsSummary() (block int64, audit int64, err error) {
	gormx.GetDB().Model(&model.WAFEvent{}).Where("action = ?", "block").Count(&block)
	gormx.GetDB().Model(&model.WAFEvent{}).Where("action = ?", "audit").Count(&audit)
	return block, audit, nil
}

// ClearDemoWAFEvents removes rows seeded for admin-console demos (waf-demo.example.com).
func (a *Audit) ClearDemoWAFEvents() (int64, error) {
	res := gormx.GetDB().Where("host = ?", "waf-demo.example.com").Delete(&model.WAFEvent{})
	return res.RowsAffected, res.Error
}

// pruneOldWAFEvents deletes events older than the given duration string (e.g. "720h").
func (a *Audit) PruneOldWAFEvents(olderThan string) (int64, error) {
	dur, err := time.ParseDuration(olderThan)
	if err != nil {
		return 0, err
	}
	cutoff := time.Now().Add(-dur)
	res := gormx.GetDB().Where("created_at < ?", cutoff).Delete(&model.WAFEvent{})
	return res.RowsAffected, res.Error
}
