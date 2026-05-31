package service

import (
	"strings"
	"time"

	"github.com/go-zoox/gormx"
	"github.com/go-zoox/ingress/core/admin/model"
)

const (
	parseIssueStatusOpen     = "open"
	parseIssueStatusIgnored  = "ignored"
	parseIssueStatusResolved = "resolved"
)

// AccessLogParseIssueRow is one persisted parse issue for the admin UI.
type AccessLogParseIssueRow struct {
	ID          uint      `json:"id"`
	Fingerprint string    `json:"fingerprint"`
	SampleLine  string    `json:"sample_line"`
	Reason      string    `json:"reason"`
	HitCount    int       `json:"hit_count"`
	Status      string    `json:"status"`
	FirstSeenAt time.Time `json:"first_seen_at"`
	LastSeenAt  time.Time `json:"last_seen_at"`
	Note        string    `json:"note,omitempty"`
}

// ParseIssues persists and lists access.log parse failures.
type ParseIssues struct{}

func NewParseIssues() *ParseIssues {
	return &ParseIssues{}
}

func (p *ParseIssues) RecordCandidates(candidates []ParseIssueCandidate) error {
	if len(candidates) == 0 {
		return nil
	}
	now := time.Now()
	db := gormx.GetDB()
	for _, c := range candidates {
		if strings.TrimSpace(c.Fingerprint) == "" {
			continue
		}
		var row model.AccessLogParseIssue
		err := db.Where("fingerprint = ?", c.Fingerprint).First(&row).Error
		if err != nil {
			row = model.AccessLogParseIssue{
				Fingerprint: c.Fingerprint,
				SampleLine:  c.Line,
				Reason:      c.Reason,
				HitCount:    1,
				Status:      parseIssueStatusOpen,
				FirstSeenAt: now,
				LastSeenAt:  now,
			}
			if createErr := db.Create(&row).Error; createErr != nil {
				return createErr
			}
			continue
		}
		updates := map[string]any{
			"hit_count":    row.HitCount + 1,
			"last_seen_at": now,
		}
		if row.Status == parseIssueStatusResolved {
			updates["status"] = parseIssueStatusOpen
			updates["resolved_at"] = nil
			updates["note"] = ""
		}
		if strings.TrimSpace(row.SampleLine) == "" && c.Line != "" {
			updates["sample_line"] = c.Line
		}
		if strings.TrimSpace(row.Reason) == "" && c.Reason != "" {
			updates["reason"] = c.Reason
		}
		if err := db.Model(&row).Updates(updates).Error; err != nil {
			return err
		}
	}
	return nil
}

func (p *ParseIssues) List(status string, limit int) ([]AccessLogParseIssueRow, error) {
	if limit <= 0 {
		limit = 20
	}
	var rows []model.AccessLogParseIssue
	q := gormx.GetDB().Model(&model.AccessLogParseIssue{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if err := q.Order("last_seen_at desc, id desc").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]AccessLogParseIssueRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, toParseIssueRow(row))
	}
	return out, nil
}

func (p *ParseIssues) OpenCount() (int64, error) {
	var count int64
	err := gormx.GetDB().Model(&model.AccessLogParseIssue{}).Where("status = ?", parseIssueStatusOpen).Count(&count).Error
	return count, err
}

func (p *ParseIssues) SetStatus(id uint, status, note string) (*AccessLogParseIssueRow, error) {
	status = strings.TrimSpace(status)
	switch status {
	case parseIssueStatusIgnored, parseIssueStatusResolved, parseIssueStatusOpen:
	default:
		status = parseIssueStatusIgnored
	}
	var row model.AccessLogParseIssue
	if err := gormx.GetDB().First(&row, id).Error; err != nil {
		return nil, err
	}
	now := time.Now()
	updates := map[string]any{
		"status":       status,
		"note":         strings.TrimSpace(note),
		"last_seen_at": now,
	}
	if status == parseIssueStatusResolved {
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
	out := toParseIssueRow(row)
	return &out, nil
}

// BatchSetStatus updates many parse issues to the same status and optional note.
func (p *ParseIssues) BatchSetStatus(ids []uint, status, note string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	status = strings.TrimSpace(status)
	switch status {
	case parseIssueStatusIgnored, parseIssueStatusResolved, parseIssueStatusOpen:
	default:
		status = parseIssueStatusIgnored
	}
	now := time.Now()
	updates := map[string]any{
		"status":       status,
		"note":         strings.TrimSpace(note),
		"last_seen_at": now,
	}
	if status == parseIssueStatusResolved {
		updates["resolved_at"] = now
	} else {
		updates["resolved_at"] = nil
	}
	res := gormx.GetDB().Model(&model.AccessLogParseIssue{}).Where("id IN ?", ids).Updates(updates)
	return res.RowsAffected, res.Error
}

// CountByStatus counts parse issues for a triage status (open, resolved, ignored).
func (p *ParseIssues) CountByStatus(status string) (int64, error) {
	var count int64
	q := gormx.GetDB().Model(&model.AccessLogParseIssue{})
	if strings.TrimSpace(status) != "" {
		q = q.Where("status = ?", status)
	}
	err := q.Count(&count).Error
	return count, err
}

// SetAllOpenStatus updates every open parse issue (not only the current UI page).
func (p *ParseIssues) SetAllOpenStatus(status, note string) (int64, error) {
	status = strings.TrimSpace(status)
	switch status {
	case parseIssueStatusIgnored, parseIssueStatusResolved, parseIssueStatusOpen:
	default:
		status = parseIssueStatusIgnored
	}
	now := time.Now()
	updates := map[string]any{
		"status":       status,
		"note":         strings.TrimSpace(note),
		"last_seen_at": now,
	}
	if status == parseIssueStatusResolved {
		updates["resolved_at"] = now
	} else {
		updates["resolved_at"] = nil
	}
	res := gormx.GetDB().Model(&model.AccessLogParseIssue{}).
		Where("status = ?", parseIssueStatusOpen).
		Updates(updates)
	return res.RowsAffected, res.Error
}

func toParseIssueRow(row model.AccessLogParseIssue) AccessLogParseIssueRow {
	return AccessLogParseIssueRow{
		ID:          row.ID,
		Fingerprint: row.Fingerprint,
		SampleLine:  row.SampleLine,
		Reason:      row.Reason,
		HitCount:    row.HitCount,
		Status:      row.Status,
		FirstSeenAt: row.FirstSeenAt,
		LastSeenAt:  row.LastSeenAt,
		Note:        row.Note,
	}
}

// AccessLogParseIssueDetail is the drawer payload for one parse issue.
type AccessLogParseIssueDetail struct {
	AccessLogParseIssueRow
	Diagnosis ParseDiagnosis  `json:"diagnosis"`
	Context   []LogContextEntry `json:"context"`
}

// GetDetail loads one parse issue with diagnosis and surrounding log context.
func (p *ParseIssues) GetDetail(id uint, logs *Logs) (*AccessLogParseIssueDetail, error) {
	var row model.AccessLogParseIssue
	if err := gormx.GetDB().First(&row, id).Error; err != nil {
		return nil, err
	}
	line := row.SampleLine
	if strings.TrimSpace(line) == "" {
		line = row.Fingerprint
	}
	diag := DiagnoseAccessLogLine(line)
	ctx, err := logs.AccessLogContextForFingerprint(row.Fingerprint, 3, 3)
	if err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = []LogContextEntry{}
	}
	out := &AccessLogParseIssueDetail{
		AccessLogParseIssueRow: toParseIssueRow(row),
		Diagnosis:              diag,
		Context:                ctx,
	}
	return out, nil
}
