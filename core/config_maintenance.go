package core

import (
	"strings"

	"github.com/go-zoox/ingress/core/service"
)

// MaintenanceConfig is the global maintenance host registry and default response settings.
type MaintenanceConfig struct {
	Hosts          service.MaintenanceHostList         `config:"hosts"`
	RetryAfter     int64                               `config:"retry_after"`
	Title          string                              `config:"title"`
	Subtitle       string                              `config:"subtitle"`
	Bypass         service.MaintenanceBypass           `config:"bypass"`
	ResponseHeader service.MaintenanceResponseHeader   `config:"response_header"`
	// StatusPath is the JSON maintenance status probe path (default /_/ingress/status).
	StatusPath string `config:"status_path"`
}

func (m MaintenanceConfig) Configured() bool {
	return len(m.Hosts) > 0 ||
		m.RetryAfter > 0 ||
		m.Title != "" ||
		m.Subtitle != "" ||
		len(m.Bypass.AllowIPs) > 0 ||
		len(m.Bypass.Paths) > 0 ||
		m.Bypass.Header.Name != "" ||
		m.Bypass.Header.Value != "" ||
		m.ResponseHeader.Configured() ||
		strings.TrimSpace(m.StatusPath) != ""
}
