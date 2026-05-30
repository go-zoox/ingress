package jobs

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-zoox/gormx"
	"github.com/go-zoox/ingress/core/admin/model"
)

// RunRow is one execution history entry for the API.
type RunRow struct {
	ID            uint             `json:"id"`
	JobID         string           `json:"job_id"`
	Source        string           `json:"source"`
	Kind          string           `json:"kind"`
	Status        string           `json:"status"`
	DurationMs    float64          `json:"duration_ms"`
	OutputPreview string           `json:"output_preview,omitempty"`
	Result        *RunResultDetail `json:"result,omitempty"`
	Error         string           `json:"error,omitempty"`
	Trigger       string           `json:"trigger"`
	StartedAt     time.Time        `json:"started_at"`
	FinishedAt    time.Time        `json:"finished_at"`
}

const runHistoryDefaultLimit = 50

func createRun(jobID, source, kind, trigger string) (*model.JobRun, error) {
	now := time.Now()
	row := &model.JobRun{
		JobID:     jobID,
		Source:    source,
		Kind:      kind,
		Status:    "running",
		Trigger:   trigger,
		StartedAt: now,
	}
	if err := gormx.GetDB().Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

func finishRun(row *model.JobRun, status string, durationMs float64, preview, errMsg string, detail RunResultDetail) error {
	if row == nil {
		return fmt.Errorf("job run row is nil")
	}
	row.Status = status
	row.DurationMs = durationMs
	row.OutputPreview = preview
	row.Error = errMsg
	row.FinishedAt = time.Now()
	if b, err := json.Marshal(detail); err == nil {
		row.ResultDetail = string(b)
	}
	return gormx.GetDB().Save(row).Error
}

func listRuns(jobID string, limit int, includeResult bool) ([]RunRow, error) {
	if limit <= 0 {
		limit = runHistoryDefaultLimit
	}
	var rows []model.JobRun
	q := gormx.GetDB().Order("started_at desc, id desc").Limit(limit)
	if jobID != "" {
		q = q.Where("job_id = ?", jobID)
	}
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]RunRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToRunRow(row, includeResult))
	}
	return out, nil
}

func getRun(id uint) (*RunRow, error) {
	var row model.JobRun
	if err := gormx.GetDB().First(&row, id).Error; err != nil {
		return nil, err
	}
	r := rowToRunRow(row, true)
	return &r, nil
}

func rowToRunRow(row model.JobRun, includeResult bool) RunRow {
	r := RunRow{
		ID:            row.ID,
		JobID:         row.JobID,
		Source:        row.Source,
		Kind:          row.Kind,
		Status:        row.Status,
		DurationMs:    row.DurationMs,
		OutputPreview: row.OutputPreview,
		Error:         row.Error,
		Trigger:       row.Trigger,
		StartedAt:     row.StartedAt,
		FinishedAt:    row.FinishedAt,
	}
	if includeResult && strings.TrimSpace(row.ResultDetail) != "" {
		var detail RunResultDetail
		if err := json.Unmarshal([]byte(row.ResultDetail), &detail); err == nil {
			r.Result = &detail
		}
	}
	return r
}

func lastRunForJob(jobID string) (*RunRow, error) {
	rows, err := listRuns(jobID, 1, false)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}
