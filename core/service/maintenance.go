package service

import "strings"

const (
	MaintenanceScopeAll    = "all"
	MaintenanceScopeListed = "listed"
)

// MaintenanceWindow limits maintenance to a time range (RFC3339). Both start and end are required when maintenance is enabled.
type MaintenanceWindow struct {
	Start string `config:"start"`
	End   string `config:"end"`
}

func (w MaintenanceWindow) Configured() bool {
	return strings.TrimSpace(w.Start) != "" || strings.TrimSpace(w.End) != ""
}

// Maintenance puts a host-level upstream into planned downtime mode.
// Only rules[].backend.service may configure this (not path backends or fallback).
type Maintenance struct {
	Enabled bool `config:"enabled"`
	// Scope: all (default) applies to every host matched by the rule; listed applies only to maintenance.hosts on the rule.
	Scope string `config:"scope,default=all"`
	// Hosts is required when scope is listed; each entry must set hosts[].window (start and end).
	Hosts MaintenanceHostList `config:"hosts"`
	// Window is required when scope is all and maintenance is enabled.
	Window MaintenanceWindow `config:"window"`
	// RetryAfter sets Retry-After response header (seconds).
	RetryAfter int64 `config:"retry_after"`
	// Title overrides the built-in 503 heading when non-empty.
	Title string `config:"title"`
	// Subtitle overrides the built-in 503 message when non-empty.
	Subtitle string `config:"subtitle"`
	// ResponseHeader is sent on maintenance 503 and /_/ingress/status when active (defaults: X-Ingress-Maintenance / 1).
	ResponseHeader MaintenanceResponseHeader `config:"response_header"`
	Bypass         MaintenanceBypass         `config:"bypass"`
}

// MaintenanceResponseHeader identifies maintenance responses for clients and probes.
type MaintenanceResponseHeader struct {
	Name  string `config:"name"`
	Value string `config:"value"`
}

func (h MaintenanceResponseHeader) Configured() bool {
	return strings.TrimSpace(h.Name) != "" || strings.TrimSpace(h.Value) != ""
}

// MaintenanceBypass allows selected requests through while maintenance is active.
type MaintenanceBypass struct {
	AllowIPs []string `config:"allow_ips"`
	Paths    []string `config:"paths"`
	Header   MaintenanceBypassHeader `config:"header"`
}

// MaintenanceBypassHeader matches a request header name/value pair.
type MaintenanceBypassHeader struct {
	Name  string `config:"name"`
	Value string `config:"value"`
}

func (m Maintenance) Configured() bool {
	return m.Enabled ||
		m.RetryAfter > 0 ||
		m.Title != "" ||
		m.Subtitle != "" ||
		len(m.Hosts) > 0 ||
		len(m.Bypass.AllowIPs) > 0 ||
		len(m.Bypass.Paths) > 0 ||
		m.Bypass.Header.Name != "" ||
		m.Bypass.Header.Value != "" ||
		m.ResponseHeader.Configured()
}

func (m Maintenance) EffectiveScope() string {
	s := strings.ToLower(strings.TrimSpace(m.Scope))
	if s == "" {
		return MaintenanceScopeAll
	}
	return s
}
