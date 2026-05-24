package handler

import (
	"github.com/go-zoox/ingress/core/admin/service"
	"github.com/go-zoox/zoox"
)

// HealthHandler serves health-check results.
type HealthHandler struct {
	health *service.HealthCheckService
}

// NewHealthHandler creates a new health check handler.
func NewHealthHandler(health *service.HealthCheckService) *HealthHandler {
	return &HealthHandler{health: health}
}

// ListChecks handles GET /api/v1/healthcheck
func (h *HealthHandler) ListChecks(ctx *zoox.Context) {
	if h.health == nil {
		ok(ctx, zoox.H{
			"checks":  []interface{}{},
			"summary": service.HealthSummary{},
		})
		return
	}

	checks, summary := h.health.ListResults()
	ok(ctx, zoox.H{
		"checks":  checks,
		"summary": summary,
	})
}
