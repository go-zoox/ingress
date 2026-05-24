package service

import (
	"time"

	"github.com/go-zoox/gormx"
	"github.com/go-zoox/ingress/core/admin/model"
)

// WAFAuditFilter holds optional filters for listing WAF events.
type WAFAuditFilter struct {
	Action    string
	Host      string
	ClientIP  string
	Rule      string
	Path      string // added for frontend path filter
	TimeStart string // RFC3339 or "2006-01-02"
	TimeEnd   string
	Limit     int
}

// Audit writes admin audit rows.
type Audit struct{}

func NewAudit() *Audit {
	return &Audit{}
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

func (a *Audit) RecordWAFEvent(action, rule, host, path, clientIP string) error {
	row := &model.WAFEvent{
		Action:    action,
		Rule:      rule,
		Host:      host,
		Path:      path,
		ClientIP:  clientIP,
		CreatedAt: time.Now(),
	}
	return gormx.GetDB().Create(row).Error
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
	if f.TimeStart != "" {
		q = q.Where("created_at >= ?", f.TimeStart)
	}
	if f.TimeEnd != "" {
		q = q.Where("created_at <= ?", f.TimeEnd)
	}
	err := q.Order("created_at desc").Limit(f.Limit).Find(&rows).Error
	return rows, err
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
