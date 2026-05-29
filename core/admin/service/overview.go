package service

import (
	"github.com/go-zoox/ingress/core/admin/model"
)

// OverviewSnapshot aggregates dashboard data for the admin overview page.
type OverviewSnapshot struct {
	Window        string                   `json:"window"`
	Status        OverviewStatus           `json:"status"`
	Metrics       OverviewMetrics          `json:"metrics"`
	System        SystemMetricsSnapshot    `json:"system"`
	Certs         []TLSCertRow             `json:"certs"`
	HealthChecks  []HealthCheckResult      `json:"health_checks"`
	HealthSummary HealthSummary            `json:"health_summary"`
	WAFBlocks     []model.WAFEvent         `json:"waf_blocks"`
	ParseIssues   []AccessLogParseIssueRow `json:"parse_issues"`
	Revisions     []ConfigRevisionSummary  `json:"revisions"`
}

// OverviewBuilder assembles overview snapshots from admin services.
type OverviewBuilder struct {
	ingress     *Ingress
	metrics     *Metrics
	system      *SystemMetrics
	tls         *TLS
	health      *HealthCheckService
	audit       *Audit
	parseIssues *ParseIssues
	config      *Config
}

// NewOverviewBuilder creates a snapshot builder wired to admin services.
func NewOverviewBuilder(
	ingress *Ingress,
	metrics *Metrics,
	system *SystemMetrics,
	tls *TLS,
	health *HealthCheckService,
	audit *Audit,
	parseIssues *ParseIssues,
	config *Config,
) *OverviewBuilder {
	return &OverviewBuilder{
		ingress:     ingress,
		metrics:     metrics,
		system:      system,
		tls:         tls,
		health:      health,
		audit:       audit,
		parseIssues: parseIssues,
		config:      config,
	}
}

// Snapshot builds the aggregated overview payload for the given metrics window.
func (b *OverviewBuilder) Snapshot(window string) OverviewSnapshot {
	if b == nil {
		return OverviewSnapshot{
			Window:       normalizeMetricsWindow(window),
			Certs:        []TLSCertRow{},
			WAFBlocks:    []model.WAFEvent{},
			ParseIssues:  []AccessLogParseIssueRow{},
			Revisions:    []ConfigRevisionSummary{},
			HealthChecks: []HealthCheckResult{},
		}
	}
	window = normalizeMetricsWindow(window)

	snap := OverviewSnapshot{
		Window:       window,
		Certs:        []TLSCertRow{},
		WAFBlocks:    []model.WAFEvent{},
		ParseIssues:  []AccessLogParseIssueRow{},
		Revisions:    []ConfigRevisionSummary{},
		HealthChecks: []HealthCheckResult{},
	}

	if b.metrics != nil {
		snap.Metrics = b.metrics.Overview(window)
	}
	if b.system != nil {
		snap.System = b.system.Snapshot(window)
	}
	if b.tls != nil {
		if rows, err := b.tls.List(); err == nil {
			snap.Certs = rows
		}
		if snap.Certs == nil {
			snap.Certs = []TLSCertRow{}
		}
	}
	if b.health != nil {
		snap.HealthChecks, snap.HealthSummary = b.health.ListResults()
	}
	if b.audit != nil {
		rows, err := b.audit.ListWAFEvents(WAFAuditFilter{
			Action: "block",
			Status: "open",
			Limit:  30,
		})
		if err == nil {
			snap.WAFBlocks = overviewWAFBlocks(rows, 8)
		}
		if snap.WAFBlocks == nil {
			snap.WAFBlocks = []model.WAFEvent{}
		}
	}
	if b.parseIssues != nil {
		if rows, err := b.parseIssues.List("open", 10); err == nil {
			snap.ParseIssues = rows
		}
		if snap.ParseIssues == nil {
			snap.ParseIssues = []AccessLogParseIssueRow{}
		}
	}
	if b.config != nil {
		if rows, err := b.config.ListRevisions(5); err == nil {
			snap.Revisions = rows
		}
		if snap.Revisions == nil {
			snap.Revisions = []ConfigRevisionSummary{}
		}
	}
	if b.ingress != nil {
		snap.Status = BuildOverviewStatus(b.ingress, b.config)
	}

	return snap
}

func overviewWAFBlocks(rows []model.WAFEvent, limit int) []model.WAFEvent {
	if limit <= 0 {
		limit = 8
	}
	out := make([]model.WAFEvent, 0, limit)
	for _, row := range rows {
		if row.Action != "block" {
			continue
		}
		out = append(out, row)
		if len(out) >= limit {
			break
		}
	}
	return out
}
