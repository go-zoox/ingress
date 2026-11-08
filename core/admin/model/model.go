package model

import "time"

// MigrateModels returns models for gorm AutoMigrate.
func MigrateModels() []any {
	return []any{
		&AuditLog{},
		&WAFEvent{},
		&ConfigRevision{},
		&AccessLogParseIssue{},
		&JobRun{},
		&MetricsMinuteBucket{},
		&WAFBan{},
		&RBACPermission{},
		&RBACRole{},
		&RBACUser{},
	}
}

// AuditLog records admin mutations (save, reload, login).
type AuditLog struct {
	ID        uint      `gorm:"primaryKey"`
	Action    string    `gorm:"size:64;index"`
	Detail    string    `gorm:"type:text"`
	Actor     string    `gorm:"size:128"`
	CreatedAt time.Time `gorm:"index"`
}

// WAFEvent stores WAF block/audit events (ingested from logs or live hooks later).
type WAFEvent struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	Action     string     `gorm:"size:16;index" json:"action"` // block | audit
	Rule       string     `gorm:"size:128" json:"rule"`
	Host       string     `gorm:"size:255;index" json:"host"`
	Path       string     `gorm:"type:text" json:"path"`
	ClientIP   string     `gorm:"size:64" json:"client_ip"`
	UserAgent  string     `gorm:"size:512" json:"user_agent"`
	Status     string     `gorm:"size:16;index" json:"status"` // open | ignored | resolved
	Note       string     `gorm:"size:512" json:"note,omitempty"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
	CreatedAt  time.Time  `gorm:"index" json:"created_at"`
}

// WAFBan stores a runtime ban entry (operational state, not ingress.yaml config).
//
// Bans deliberately live in the admin database rather than the ingress config
// file: they are temporary, have a lifetime, and must apply without a config
// reload (a reload would flush the shared cache and reset rate limiters).
type WAFBan struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Kind      string     `gorm:"size:16;index" json:"kind"` // ip | ip_cidr | ua_contains | ua_regex
	Value     string     `gorm:"size:255;index" json:"value"`
	Hosts     string     `gorm:"type:text" json:"-"` // JSON array; empty = global
	Reason    string     `gorm:"size:512" json:"reason"`
	Source    string     `gorm:"size:16" json:"source"` // manual | auto
	Actor     string     `gorm:"size:128" json:"actor"`
	Hits      int64      `gorm:"default:0" json:"hits"`
	Enabled   bool       `gorm:"index" json:"enabled"`
	ExpiresAt *time.Time `gorm:"index" json:"expires_at"`
	CreatedAt time.Time  `gorm:"index" json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// ConfigRevision stores published YAML snapshots.
type ConfigRevision struct {
	ID        uint      `gorm:"primaryKey"`
	Hash      string    `gorm:"size:32;index"`
	Content   string    `gorm:"type:text"`
	Note      string    `gorm:"size:255"`
	CreatedAt time.Time `gorm:"index"`
}
