package service

// EventsTabSummary is aggregate counts for the admin events page tabs.
type EventsTabSummary struct {
	WAFBlock    int64 `json:"waf_block"`
	ParseIssues int64 `json:"parse_issues"`
	HealthDown  int64 `json:"health_down"`
	TLSWarn     int64 `json:"tls_warn"`
	Total       int64 `json:"total"`
}

// BuildEventsTabSummary aggregates DB-backed counts for one triage tab.
func BuildEventsTabSummary(status string, wafBlock, parseIssues, healthDown, tlsWarn int64) EventsTabSummary {
	out := EventsTabSummary{
		WAFBlock:    wafBlock,
		ParseIssues: parseIssues,
		HealthDown:  healthDown,
		TLSWarn:     tlsWarn,
	}
	switch status {
	case "resolved", "ignored":
		out.Total = wafBlock + parseIssues
	default:
		out.Total = wafBlock + parseIssues + healthDown + tlsWarn
	}
	return out
}
