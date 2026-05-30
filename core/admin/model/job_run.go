package model

import "time"

// JobRun stores one execution record for a scheduled job.
type JobRun struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	JobID          string    `gorm:"size:128;index" json:"job_id"`
	Source         string    `gorm:"size:16;index" json:"source"` // builtin | config
	Kind           string    `gorm:"size:32" json:"kind"`
	Status         string    `gorm:"size:16;index" json:"status"` // running | success | failed
	DurationMs     float64   `json:"duration_ms"`
	OutputPreview  string    `gorm:"type:text" json:"output_preview,omitempty"`
	ResultDetail   string    `gorm:"type:text" json:"-"`
	Error          string    `gorm:"type:text" json:"error,omitempty"`
	Trigger        string    `gorm:"size:16" json:"trigger"` // schedule | manual
	StartedAt      time.Time `gorm:"index" json:"started_at"`
	FinishedAt     time.Time `json:"finished_at"`
}
