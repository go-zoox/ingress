package service

import (
	"encoding/json"
	"reflect"

	"github.com/go-zoox/ingress/core/admin/model"
)

// OverviewSSEPatch is an incremental SSE update after the initial overview snapshot.
// Object sections (status, metrics, system, health_summary) contain only changed fields.
type OverviewSSEPatch struct {
	Window        string                   `json:"window,omitempty"`
	Seq           int64                    `json:"seq"`
	Status        json.RawMessage          `json:"status,omitempty"`
	Metrics       json.RawMessage          `json:"metrics,omitempty"`
	System        json.RawMessage          `json:"system,omitempty"`
	HealthSummary json.RawMessage          `json:"health_summary,omitempty"`
	Certs         []TLSCertRow             `json:"certs,omitempty"`
	HealthChecks  []HealthCheckResult      `json:"health_checks,omitempty"`
	WAFBlocks     []model.WAFEvent         `json:"waf_blocks,omitempty"`
	ParseIssues   []AccessLogParseIssueRow `json:"parse_issues,omitempty"`
	Revisions     []ConfigRevisionSummary  `json:"revisions,omitempty"`
}

func (p OverviewSSEPatch) isEmpty() bool {
	return p.Window == "" &&
		len(p.Status) == 0 &&
		len(p.Metrics) == 0 &&
		len(p.System) == 0 &&
		len(p.HealthSummary) == 0 &&
		p.Certs == nil &&
		p.HealthChecks == nil &&
		p.WAFBlocks == nil &&
		p.ParseIssues == nil &&
		p.Revisions == nil
}

func computeOverviewSSEPatch(prev, next OverviewSnapshot) OverviewSSEPatch {
	var p OverviewSSEPatch
	if prev.Window != next.Window {
		p.Window = next.Window
	}
	if raw, ok := patchObject(statusForPatchCompare(prev.Status), statusForPatchCompare(next.Status)); ok {
		p.Status = raw
	}
	if raw, ok := patchObject(prev.Metrics, next.Metrics); ok {
		p.Metrics = raw
	}
	if raw, ok := patchObject(prev.System, next.System); ok {
		p.System = raw
	}
	if raw, ok := patchObject(prev.HealthSummary, next.HealthSummary); ok {
		p.HealthSummary = raw
	}
	if !reflect.DeepEqual(prev.Certs, next.Certs) {
		p.Certs = append([]TLSCertRow(nil), next.Certs...)
	}
	if !reflect.DeepEqual(prev.HealthChecks, next.HealthChecks) {
		p.HealthChecks = append([]HealthCheckResult(nil), next.HealthChecks...)
	}
	if !reflect.DeepEqual(prev.WAFBlocks, next.WAFBlocks) {
		p.WAFBlocks = append([]model.WAFEvent(nil), next.WAFBlocks...)
	}
	if !reflect.DeepEqual(prev.ParseIssues, next.ParseIssues) {
		p.ParseIssues = append([]AccessLogParseIssueRow(nil), next.ParseIssues...)
	}
	if !reflect.DeepEqual(prev.Revisions, next.Revisions) {
		p.Revisions = append([]ConfigRevisionSummary(nil), next.Revisions...)
	}
	if !p.isEmpty() {
		p.Window = next.Window
	}
	return p
}

func applyOverviewSSEPatch(base OverviewSnapshot, patch OverviewSSEPatch) OverviewSnapshot {
	out := base
	if patch.Window != "" {
		out.Window = patch.Window
	}
	if len(patch.Status) > 0 {
		out.Status = applyJSONPatch(out.Status, patch.Status)
	}
	if len(patch.Metrics) > 0 {
		out.Metrics = applyJSONPatch(out.Metrics, patch.Metrics)
	}
	if len(patch.System) > 0 {
		out.System = applyJSONPatch(out.System, patch.System)
	}
	if len(patch.HealthSummary) > 0 {
		out.HealthSummary = applyJSONPatch(out.HealthSummary, patch.HealthSummary)
	}
	if patch.Certs != nil {
		out.Certs = patch.Certs
	}
	if patch.HealthChecks != nil {
		out.HealthChecks = patch.HealthChecks
	}
	if patch.WAFBlocks != nil {
		out.WAFBlocks = patch.WAFBlocks
	}
	if patch.ParseIssues != nil {
		out.ParseIssues = patch.ParseIssues
	}
	if patch.Revisions != nil {
		out.Revisions = patch.Revisions
	}
	return out
}

func statusForPatchCompare(s OverviewStatus) OverviewStatus {
	s.LastReload = ""
	return s
}

// patchObject returns a JSON object containing only top-level keys that changed.
func patchObject(prev, next any) (json.RawMessage, bool) {
	if reflect.DeepEqual(prev, next) {
		return nil, false
	}
	prevMap, err := valueToMap(prev)
	if err != nil {
		return nil, false
	}
	nextMap, err := valueToMap(next)
	if err != nil {
		return nil, false
	}
	diff := make(map[string]any)
	for key, nextVal := range nextMap {
		prevVal, ok := prevMap[key]
		if !ok || !reflect.DeepEqual(prevVal, nextVal) {
			diff[key] = nextVal
		}
	}
	if len(diff) == 0 {
		return nil, false
	}
	raw, err := json.Marshal(diff)
	if err != nil {
		return nil, false
	}
	return raw, true
}

func valueToMap(v any) (map[string]any, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]any{}
	}
	return out, nil
}

func applyJSONPatch[T any](base T, patch json.RawMessage) T {
	data, err := json.Marshal(base)
	if err != nil {
		return base
	}
	var merged map[string]any
	if err := json.Unmarshal(data, &merged); err != nil {
		return base
	}
	var patchMap map[string]any
	if err := json.Unmarshal(patch, &patchMap); err != nil {
		return base
	}
	for key, val := range patchMap {
		merged[key] = val
	}
	outData, err := json.Marshal(merged)
	if err != nil {
		return base
	}
	var out T
	if err := json.Unmarshal(outData, &out); err != nil {
		return base
	}
	return out
}
