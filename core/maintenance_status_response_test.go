package core

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/go-zoox/ingress/core/service"
)

func TestCompileMaintenanceStatusResponse_ValidTemplates(t *testing.T) {
	t.Parallel()
	compiled, err := compileMaintenanceStatusResponse(service.MaintenanceStatusResponse{
		OK:          `{"ready":true,"host":"${host}"}`,
		Maintenance: `{"ready":false,"message":"${title}","retry_after":${retry_after}}`,
		ContentType: "application/json; charset=utf-8",
	}, "maintenance.status_response")
	if err != nil {
		t.Fatal(err)
	}
	if !compiled.useCustomOK || !compiled.useCustomMaintenance {
		t.Fatalf("expected custom templates enabled, got %+v", compiled)
	}
	if compiled.contentType != "application/json; charset=utf-8" {
		t.Fatalf("unexpected content type %q", compiled.contentType)
	}
}

func TestCompileMaintenanceStatusResponse_InvalidJSON(t *testing.T) {
	t.Parallel()
	_, err := compileMaintenanceStatusResponse(service.MaintenanceStatusResponse{
		OK: `{not json}`,
	}, "maintenance.status_response")
	if err == nil || !strings.Contains(err.Error(), "valid JSON") {
		t.Fatalf("expected JSON validation error, got %v", err)
	}
}

func TestExpandMaintenanceStatusResponseTemplate(t *testing.T) {
	t.Parallel()
	raw := `{"host":"${host}","title":"${title}","retry_after":${retry_after},"status":"${status}"}`
	got, err := expandMaintenanceStatusResponseTemplate(raw, maintenanceStatusTemplateContext{
		Hostname:   "app.example.com",
		Title:      `Say "hello"`,
		RetryAfter: 120,
		Status:     "maintenance",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(got)) {
		t.Fatalf("expected valid JSON, got %q", got)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(got), &body); err != nil {
		t.Fatal(err)
	}
	if body["host"] != "app.example.com" {
		t.Fatalf("unexpected host: %v", body["host"])
	}
	if body["title"] != `Say "hello"` {
		t.Fatalf("unexpected title: %v", body["title"])
	}
	if body["retry_after"] != float64(120) {
		t.Fatalf("unexpected retry_after: %v", body["retry_after"])
	}
	if body["status"] != "maintenance" {
		t.Fatalf("unexpected status: %v", body["status"])
	}
}

func TestRenderIngressStatusBody_CustomTemplates(t *testing.T) {
	t.Parallel()
	ins := &core{
		globalMaintenance: compiledGlobalMaintenance{
			statusResponse: compiledMaintenanceStatusResponse{
				okBody:               `{"ready":true,"host":"${host}"}`,
				maintenanceBody:      `{"ready":false,"message":"${title}","retry_after":${retry_after}}`,
				contentType:          "application/json; charset=utf-8",
				useCustomOK:          true,
				useCustomMaintenance: true,
			},
		},
	}
	settings := compiledMaintenanceSettings{
		Title:      "Planned downtime",
		RetryAfter: 300,
		responseHeader: compiledMaintenanceResponseHeader{
			name:  headerXIngressMaintenance,
			value: ingressMaintenanceHeaderVal,
		},
	}

	okBody, err := ins.renderIngressStatusBody(false, settings, compiledMaintenanceWindow{}, "app.example.com")
	if err != nil {
		t.Fatal(err)
	}
	var ok map[string]any
	if err := json.Unmarshal([]byte(okBody), &ok); err != nil {
		t.Fatal(err)
	}
	if ok["ready"] != true || ok["host"] != "app.example.com" {
		t.Fatalf("unexpected ok body: %v", ok)
	}

	maintBody, err := ins.renderIngressStatusBody(true, settings, compiledMaintenanceWindow{}, "app.example.com")
	if err != nil {
		t.Fatal(err)
	}
	var maint map[string]any
	if err := json.Unmarshal([]byte(maintBody), &maint); err != nil {
		t.Fatal(err)
	}
	if maint["ready"] != false || maint["message"] != "Planned downtime" || maint["retry_after"] != float64(300) {
		t.Fatalf("unexpected maintenance body: %v", maint)
	}
}
