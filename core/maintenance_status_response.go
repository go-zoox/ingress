package core

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-zoox/ingress/core/service"
)

type compiledMaintenanceStatusResponse struct {
	okBody               string
	maintenanceBody      string
	contentType          string
	useCustomOK          bool
	useCustomMaintenance bool
}

type maintenanceStatusTemplateContext struct {
	Hostname    string
	Title       string
	Subtitle    string
	RetryAfter  int64
	HeaderName  string
	HeaderValue string
	From        string
	Until       string
	Status      string
}

func compileMaintenanceStatusResponse(r service.MaintenanceStatusResponse, loc string) (compiledMaintenanceStatusResponse, error) {
	out := compiledMaintenanceStatusResponse{
		contentType: errorPageContentTypeJSON,
	}
	if ct := strings.TrimSpace(r.ContentType); ct != "" {
		out.contentType = ct
	}
	if raw := strings.TrimSpace(r.OK); raw != "" {
		if err := validateMaintenanceStatusResponseTemplate(raw, loc+".ok", "ok"); err != nil {
			return out, err
		}
		out.okBody = raw
		out.useCustomOK = true
	}
	if raw := strings.TrimSpace(r.Maintenance); raw != "" {
		if err := validateMaintenanceStatusResponseTemplate(raw, loc+".maintenance", "maintenance"); err != nil {
			return out, err
		}
		out.maintenanceBody = raw
		out.useCustomMaintenance = true
	}
	return out, nil
}

func validateMaintenanceStatusResponseTemplate(raw, loc, status string) error {
	expanded, err := expandMaintenanceStatusResponseTemplate(raw, maintenanceStatusTemplateContext{
		Hostname:    "example.com",
		Title:       "Sample title",
		Subtitle:    "Sample subtitle",
		RetryAfter:  300,
		HeaderName:  headerXIngressMaintenance,
		HeaderValue: ingressMaintenanceHeaderVal,
		Status:      status,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", loc, err)
	}
	if !json.Valid([]byte(expanded)) {
		return fmt.Errorf("%s must be valid JSON after placeholder expansion", loc)
	}
	return nil
}

func expandMaintenanceStatusResponseTemplate(raw string, ctx maintenanceStatusTemplateContext) (string, error) {
	if raw == "" {
		return "", nil
	}
	repl := strings.NewReplacer(
		"${host}", jsonStringContent(ctx.Hostname),
		"${title}", jsonStringContent(ctx.Title),
		"${subtitle}", jsonStringContent(ctx.Subtitle),
		"${retry_after}", strconv.FormatInt(ctx.RetryAfter, 10),
		"${maintenance_header_name}", jsonStringContent(ctx.HeaderName),
		"${maintenance_header_value}", jsonStringContent(ctx.HeaderValue),
		"${maintenance_from}", jsonStringContent(ctx.From),
		"${maintenance_until}", jsonStringContent(ctx.Until),
		"${status}", jsonStringContent(ctx.Status),
	)
	return repl.Replace(raw), nil
}

func jsonStringContent(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return ""
	}
	if len(b) < 2 {
		return ""
	}
	return string(b[1 : len(b)-1])
}

func (c *core) renderIngressStatusBody(active bool, settings compiledMaintenanceSettings, window compiledMaintenanceWindow, hostname string) (string, error) {
	statusResp := c.globalMaintenance.statusResponse
	logicalStatus := "ok"
	if active {
		logicalStatus = "maintenance"
	}
	from, until := maintenanceWindowHeaderValues(window)
	tplCtx := maintenanceStatusTemplateContext{
		Hostname:    hostname,
		Title:       settings.Title,
		Subtitle:    settings.Subtitle,
		RetryAfter:  settings.RetryAfter,
		HeaderName:  settings.responseHeader.name,
		HeaderValue: settings.responseHeader.value,
		From:        from,
		Until:       until,
		Status:      logicalStatus,
	}

	if active && statusResp.useCustomMaintenance {
		return expandMaintenanceStatusResponseTemplate(statusResp.maintenanceBody, tplCtx)
	}
	if !active && statusResp.useCustomOK {
		return expandMaintenanceStatusResponseTemplate(statusResp.okBody, tplCtx)
	}

	body := ingressStatusBody{Status: logicalStatus}
	if active {
		body.Title = settings.Title
		body.Subtitle = settings.Subtitle
		body.MaintenanceHeaderName = settings.responseHeader.name
		body.MaintenanceHeaderValue = settings.responseHeader.value
		body.MaintenanceFrom = from
		body.MaintenanceUntil = until
		if settings.RetryAfter > 0 {
			body.RetryAfter = settings.RetryAfter
		}
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
