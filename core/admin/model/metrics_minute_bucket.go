package model

import "time"

// MetricsMinuteBucket stores per-minute access aggregates for admin overview persistence.
type MetricsMinuteBucket struct {
	ID              uint      `gorm:"primaryKey"`
	Minute          time.Time `gorm:"uniqueIndex;index"`
	Count           int
	S2              int
	S3              int
	S4              int
	S5              int
	WAFBlocks       int
	CacheHits       int
	DurationSumMs   float64
	DurationCount   int
	DurationMaxMs   float64
	UpstreamSumMs   float64
	UpstreamCount   int
	UpdatedAt       time.Time `gorm:"index"`
}
