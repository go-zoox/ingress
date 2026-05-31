package service

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-zoox/ingress/core/admin/model"
	"gorm.io/gorm"
)

// MigrateAccessLogOptions controls offline access.log → metrics_minute_bucket import.
type MigrateAccessLogOptions struct {
	Since  time.Time
	Until  time.Time
	DryRun bool
	// Replace deletes existing buckets in [Since, Until) before import.
	Replace bool
}

// MigrateAccessLogResult summarizes a migrate run.
type MigrateAccessLogResult struct {
	LinesRead       int
	LinesParsed     int
	LinesSkipped    int
	MinutesInserted int
	MinutesReplaced int
}

// MigrateAccessLogToBuckets streams an access log file into SQLite minute buckets.
// Each minute from the log replaces any existing bucket for that minute (idempotent re-run, not additive).
func MigrateAccessLogToBuckets(path string, opts MigrateAccessLogOptions) (MigrateAccessLogResult, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return MigrateAccessLogResult{}, fmt.Errorf("access log path is required")
	}
	f, err := os.Open(path)
	if err != nil {
		return MigrateAccessLogResult{}, err
	}
	defer f.Close()

	store := NewMetricsRollupStore()
	if store.db() == nil {
		return MigrateAccessLogResult{}, fmt.Errorf("admin database is not initialized")
	}

	if opts.Replace && !opts.Since.IsZero() && !opts.Until.IsZero() && !opts.DryRun {
		if err := store.DeleteBucketsInRange(opts.Since, opts.Until); err != nil {
			return MigrateAccessLogResult{}, err
		}
	}

	var res MigrateAccessLogResult
	pending := make(map[int64]*minuteDelta)
	flushMinute := func(key int64) error {
		d := pending[key]
		delete(pending, key)
		if d == nil || d.count <= 0 {
			return nil
		}
		minute := time.Unix(key, 0)
		if opts.DryRun {
			var existing model.MetricsMinuteBucket
			err := store.db().Where("minute = ?", minute.Truncate(time.Minute)).First(&existing).Error
			if err == nil {
				res.MinutesReplaced++
				return nil
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			res.MinutesInserted++
			return nil
		}
		replaced, err := store.ReplaceMinuteFromMigrate(minute, *d)
		if err != nil {
			return err
		}
		if replaced {
			res.MinutesReplaced++
		} else {
			res.MinutesInserted++
		}
		return nil
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		res.LinesRead++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		e, ok := ParseAccessEntry(line)
		if !ok || e.At.IsZero() {
			res.LinesSkipped++
			continue
		}
		if !opts.Since.IsZero() && e.At.Before(opts.Since) {
			continue
		}
		if !opts.Until.IsZero() && e.At.After(opts.Until) {
			continue
		}
		res.LinesParsed++
		key := e.At.Truncate(time.Minute).Unix()
		d := deltaFromEntry(e)
		if cur, ok := pending[key]; ok {
			mergeMinuteDelta(cur, d)
			continue
		}
		copy := d
		pending[key] = &copy
	}
	if err := scanner.Err(); err != nil {
		return res, err
	}
	for key := range pending {
		if err := flushMinute(key); err != nil {
			return res, err
		}
	}
	return res, nil
}

// DeleteBucketsInRange removes persisted minute buckets in [start, end).
func (s *MetricsRollupStore) DeleteBucketsInRange(start, end time.Time) error {
	db := s.db()
	if db == nil {
		return nil
	}
	if end.IsZero() {
		end = time.Now()
	}
	return db.Where("minute >= ? AND minute < ?", start.Truncate(time.Minute), end.Truncate(time.Minute)).
		Delete(&model.MetricsMinuteBucket{}).Error
}
