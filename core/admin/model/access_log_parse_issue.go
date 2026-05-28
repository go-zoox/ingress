package model

import "time"

// AccessLogParseIssue tracks access.log lines that look like ingress access entries but fail to parse.
type AccessLogParseIssue struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Fingerprint string     `gorm:"size:64;uniqueIndex" json:"fingerprint"`
	SampleLine  string     `gorm:"type:text" json:"sample_line"`
	Reason      string     `gorm:"size:64" json:"reason"`
	HitCount    int        `json:"hit_count"`
	Status      string     `gorm:"size:16;index" json:"status"` // open | ignored | resolved
	FirstSeenAt time.Time  `gorm:"index" json:"first_seen_at"`
	LastSeenAt  time.Time  `gorm:"index" json:"last_seen_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	Note        string     `gorm:"size:255" json:"note,omitempty"`
}
