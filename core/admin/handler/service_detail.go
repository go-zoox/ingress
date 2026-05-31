package handler

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/go-zoox/ingress/core/admin/service"
	"github.com/go-zoox/zoox"
)

// ServiceDetailHandler serves catalog service detail and metrics.
type ServiceDetailHandler struct {
	ingress *service.Ingress
	service *service.ServiceMetricsBuilder
}

// NewServiceDetailHandler creates a service detail handler.
func NewServiceDetailHandler(ingress *service.Ingress, svc *service.ServiceMetricsBuilder) *ServiceDetailHandler {
	return &ServiceDetailHandler{ingress: ingress, service: svc}
}

// GetDetail handles GET /api/v1/services/:name
func (h *ServiceDetailHandler) GetDetail(ctx *zoox.Context) {
	name, err := parseServiceNameParam(ctx)
	if err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}

	content, err := h.ingress.ReadYAML()
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	catalog, err := service.ParseServiceCatalog(content)
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	entry, found := service.FindCatalogService(catalog, name)
	if !found {
		fail(ctx, http.StatusNotFound, "service not found in catalog")
		return
	}

	cfg, err := h.ingress.LoadConfig()
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	refs := service.ListServiceRouteRefs(cfg, name)
	aliases := service.ServiceTargetAliases(entry, refs)

	var healthDetail zoox.H
	if entry.HealthCheck.Enable {
		healthDetail = zoox.H{
			"enabled": true,
			"method":  entry.HealthCheck.Method,
			"path":    entry.HealthCheck.Path,
			"status":  entry.HealthCheck.Status,
			"ok":      entry.HealthCheck.Ok,
		}
	}

	ok(ctx, zoox.H{
		"name":             entry.Name,
		"catalog_index":    entry.Index,
		"target":           entry.Target,
		"protocol":         entry.Protocol,
		"port":             entry.Port,
		"mode":             entry.Mode,
		"note":             entry.Note,
		"health_check":     healthDetail,
		"route_refs":       refs,
		"route_ref_count":  len(refs),
		"target_aliases":   aliases,
	})
}

// GetMetrics handles GET /api/v1/services/:name/metrics
func (h *ServiceDetailHandler) GetMetrics(ctx *zoox.Context) {
	name, err := parseServiceNameParam(ctx)
	if err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}

	content, err := h.ingress.ReadYAML()
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	catalog, err := service.ParseServiceCatalog(content)
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	entry, found := service.FindCatalogService(catalog, name)
	if !found {
		fail(ctx, http.StatusNotFound, "service not found in catalog")
		return
	}

	cfg, err := h.ingress.LoadConfig()
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	refs := service.ListServiceRouteRefs(cfg, name)
	aliases := service.ServiceTargetAliases(entry, refs)

	rangeQ, err := parseMetricsRangeQuery(ctx)
	if err != nil {
		return
	}
	windowLabel := strings.TrimSpace(ctx.Query().Get("window").String())
	if windowLabel == "" {
		windowLabel = service.WindowLabelForDuration(rangeQ.Duration())
	}

	analytics := h.service.Build(windowLabel, rangeQ, aliases)
	ok(ctx, service.ServiceAnalyticsToMap(analytics))
}

func parseServiceNameParam(ctx *zoox.Context) (string, error) {
	raw := strings.TrimSpace(ctx.Param().Get("name").String())
	if raw == "" {
		return "", &serviceNameError{}
	}
	decoded, err := url.PathUnescape(raw)
	if err != nil {
		return raw, nil
	}
	return strings.TrimSpace(decoded), nil
}

type serviceNameError struct{}

func (e *serviceNameError) Error() string {
	return "service name is required"
}
