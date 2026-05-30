package service

import "strings"

// MaintenanceStatusResponse customizes the JSON body of the maintenance status probe.
// Placeholders: ${host} ${title} ${subtitle} ${retry_after} ${maintenance_header_name}
// ${maintenance_header_value} ${status} (ok | maintenance). String placeholders expand inside JSON quotes.
type MaintenanceStatusResponse struct {
	OK          string `config:"ok"`
	Maintenance string `config:"maintenance"`
	ContentType string `config:"content_type"`
}

func (r MaintenanceStatusResponse) Configured() bool {
	return strings.TrimSpace(r.OK) != "" ||
		strings.TrimSpace(r.Maintenance) != "" ||
		strings.TrimSpace(r.ContentType) != ""
}
