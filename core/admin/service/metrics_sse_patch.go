package service

import "encoding/json"

// MetricsSSEPatch is an incremental SSE update for route/service metrics JSON.
type MetricsSSEPatch struct {
	Window string          `json:"window,omitempty"`
	Seq    int64           `json:"seq"`
	Data   json.RawMessage `json:"data,omitempty"`
}

func (p MetricsSSEPatch) isEmpty() bool {
	return p.Window == "" && len(p.Data) == 0
}

func computeMetricsSSEPatch(prev, next map[string]any) MetricsSSEPatch {
	var p MetricsSSEPatch
	prevWindow, _ := prev["window"].(string)
	nextWindow, _ := next["window"].(string)
	if prevWindow != nextWindow {
		p.Window = nextWindow
	}
	if raw, ok := patchObject(prev, next); ok {
		p.Data = raw
	}
	if !p.isEmpty() {
		p.Window = nextWindow
	}
	return p
}
