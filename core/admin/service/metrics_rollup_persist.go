package service

import (
	"errors"
	"time"

	"github.com/go-zoox/gormx"
	"github.com/go-zoox/ingress/core/admin/model"
	"gorm.io/gorm"
)

const rollupPersistRetention = 26 * time.Hour

type minuteDelta struct {
	count                   int
	s2, s3, s4, s5          int
	wafBlocks, cacheHits    int
	durationSumMs           float64
	durationCount           int
	durationMaxMs           float64
	upstreamSumMs           float64
	upstreamCount           int
}

// MetricsRollupStore persists minute aggregates to SQLite.
type MetricsRollupStore struct{}

func NewMetricsRollupStore() *MetricsRollupStore {
	return &MetricsRollupStore{}
}

func (s *MetricsRollupStore) db() *gorm.DB {
	if s == nil {
		return nil
	}
	return gormx.GetDB()
}

func (s *MetricsRollupStore) ApplyDelta(minute time.Time, d minuteDelta) error {
	db := s.db()
	if db == nil || d.count <= 0 {
		return nil
	}
	key := minute.Truncate(time.Minute)
	var row model.MetricsMinuteBucket
	err := db.Where("minute = ?", key).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row = model.MetricsMinuteBucket{
			Minute:        key,
			Count:         d.count,
			S2:            d.s2,
			S3:            d.s3,
			S4:            d.s4,
			S5:            d.s5,
			WAFBlocks:     d.wafBlocks,
			CacheHits:     d.cacheHits,
			DurationSumMs: d.durationSumMs,
			DurationCount: d.durationCount,
			DurationMaxMs: d.durationMaxMs,
			UpstreamSumMs: d.upstreamSumMs,
			UpstreamCount: d.upstreamCount,
			UpdatedAt:     time.Now(),
		}
		return db.Create(&row).Error
	}
	if err != nil {
		return err
	}
	row.Count += d.count
	row.S2 += d.s2
	row.S3 += d.s3
	row.S4 += d.s4
	row.S5 += d.s5
	row.WAFBlocks += d.wafBlocks
	row.CacheHits += d.cacheHits
	row.DurationSumMs += d.durationSumMs
	row.DurationCount += d.durationCount
	if d.durationMaxMs > row.DurationMaxMs {
		row.DurationMaxMs = d.durationMaxMs
	}
	row.UpstreamSumMs += d.upstreamSumMs
	row.UpstreamCount += d.upstreamCount
	row.UpdatedAt = time.Now()
	return db.Save(&row).Error
}

// ReplaceMinuteFromMigrate sets a minute bucket from access.log import (overwrite, not merge).
// Re-running migrate on the same log is idempotent. Returns true when an existing row was replaced.
func (s *MetricsRollupStore) ReplaceMinuteFromMigrate(minute time.Time, d minuteDelta) (bool, error) {
	db := s.db()
	if db == nil || d.count <= 0 {
		return false, nil
	}
	key := minute.Truncate(time.Minute)
	row := model.MetricsMinuteBucket{
		Minute:        key,
		Count:         d.count,
		S2:            d.s2,
		S3:            d.s3,
		S4:            d.s4,
		S5:            d.s5,
		WAFBlocks:     d.wafBlocks,
		CacheHits:     d.cacheHits,
		DurationSumMs: d.durationSumMs,
		DurationCount: d.durationCount,
		DurationMaxMs: d.durationMaxMs,
		UpstreamSumMs: d.upstreamSumMs,
		UpstreamCount: d.upstreamCount,
		UpdatedAt:     time.Now(),
	}
	var existing model.MetricsMinuteBucket
	err := db.Where("minute = ?", key).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, db.Create(&row).Error
	}
	if err != nil {
		return false, err
	}
	row.ID = existing.ID
	return true, db.Save(&row).Error
}

func (s *MetricsRollupStore) LoadSince(since time.Time) ([]model.MetricsMinuteBucket, error) {
	db := s.db()
	if db == nil {
		return nil, nil
	}
	var rows []model.MetricsMinuteBucket
	if err := db.Where("minute >= ?", since.Truncate(time.Minute)).Order("minute asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *MetricsRollupStore) LoadBetween(from, to time.Time) ([]model.MetricsMinuteBucket, error) {
	db := s.db()
	if db == nil {
		return nil, nil
	}
	from = from.Truncate(time.Minute)
	to = to.Truncate(time.Minute)
	var rows []model.MetricsMinuteBucket
	if err := db.Where("minute >= ? AND minute <= ?", from, to).Order("minute asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *MetricsRollupStore) PurgeOlderThan(days int) (int64, error) {
	db := s.db()
	if db == nil {
		return 0, nil
	}
	if days <= 0 {
		days = 30
	}
	cutoff := time.Now().AddDate(0, 0, -days)
	res := db.Where("minute < ?", cutoff).Delete(&model.MetricsMinuteBucket{})
	return res.RowsAffected, res.Error
}

func deltaFromEntry(e AccessEntry) minuteDelta {
	var d minuteDelta
	d.count = 1
	switch statusClass(e.Status) {
	case "2xx":
		d.s2 = 1
	case "3xx":
		d.s3 = 1
	case "4xx":
		d.s4 = 1
	case "5xx":
		d.s5 = 1
	}
	if e.WAFBlock {
		d.wafBlocks = 1
	}
	if e.CacheHit {
		d.cacheHits = 1
	}
	if e.DurationMs > 0 {
		d.durationSumMs = e.DurationMs
		d.durationCount = 1
		d.durationMaxMs = e.DurationMs
	}
	if e.UpstreamDurationMs > 0 {
		d.upstreamSumMs = e.UpstreamDurationMs
		d.upstreamCount = 1
	}
	return d
}

func mergeMinuteDelta(dst *minuteDelta, src minuteDelta) {
	dst.count += src.count
	dst.s2 += src.s2
	dst.s3 += src.s3
	dst.s4 += src.s4
	dst.s5 += src.s5
	dst.wafBlocks += src.wafBlocks
	dst.cacheHits += src.cacheHits
	dst.durationSumMs += src.durationSumMs
	dst.durationCount += src.durationCount
	if src.durationMaxMs > dst.durationMaxMs {
		dst.durationMaxMs = src.durationMaxMs
	}
	dst.upstreamSumMs += src.upstreamSumMs
	dst.upstreamCount += src.upstreamCount
}

func entriesFromMinuteBucket(b model.MetricsMinuteBucket) []AccessEntry {
	if b.Count <= 0 {
		return nil
	}
	minute := b.Minute.Truncate(time.Minute)
	limit := b.Count
	if limit > rollupSynthPerBucket {
		limit = rollupSynthPerBucket
	}
	if limit <= 0 {
		return nil
	}
	// Scale status mix down to the synthesis cap (totals come from bucket aggregates when needed).
	scale := float64(limit) / float64(b.Count)
	s2 := int(float64(b.S2) * scale)
	s3 := int(float64(b.S3) * scale)
	s4 := int(float64(b.S4) * scale)
	s5 := limit - s2 - s3 - s4
	if s5 < 0 {
		s5 = 0
	}
	statuses := make([]int, 0, limit)
	for i := 0; i < s2; i++ {
		statuses = append(statuses, 200)
	}
	for i := 0; i < s3; i++ {
		statuses = append(statuses, 301)
	}
	for i := 0; i < s4; i++ {
		statuses = append(statuses, 400)
	}
	for i := 0; i < s5; i++ {
		statuses = append(statuses, 500)
	}
	avgMs := 0.0
	if b.DurationCount > 0 {
		avgMs = b.DurationSumMs / float64(b.DurationCount)
	}
	upstreamAvg := 0.0
	if b.UpstreamCount > 0 {
		upstreamAvg = b.UpstreamSumMs / float64(b.UpstreamCount)
	}
	out := make([]AccessEntry, 0, len(statuses))
	for i, status := range statuses {
		at := minute.Add(time.Duration(i%60) * time.Second)
		out = append(out, AccessEntry{
			At:                 at,
			Status:             status,
			DurationMs:         avgMs,
			UpstreamDurationMs: upstreamAvg,
			CacheHit:           i < b.CacheHits,
			WAFBlock:           i < b.WAFBlocks,
		})
	}
	return out
}
