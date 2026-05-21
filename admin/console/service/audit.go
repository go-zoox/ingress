package service

import (
	"time"

	"github.com/go-zoox/gormx"
	"github.com/go-zoox/ingress/admin/console/model"
)

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

func (a *Audit) ListWAFEvents(limit int) ([]model.WAFEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows []model.WAFEvent
	err := gormx.GetDB().Order("created_at desc").Limit(limit).Find(&rows).Error
	return rows, err
}
