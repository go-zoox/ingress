package service

import (
	ingcore "github.com/go-zoox/ingress/core"
	"github.com/go-zoox/ingress/core/admin/model"
)

// FilterAccessEntriesForRoute keeps access log entries that match the route (ri/pi).
func FilterAccessEntriesForRoute(cfg *ingcore.Config, ruleIndex, pathIndex int, lines []string) []AccessEntry {
	if cfg == nil {
		return nil
	}
	var out []AccessEntry
	for _, line := range lines {
		e, ok := ParseAccessEntry(line)
		if !ok {
			continue
		}
		matched, err := ingcore.RequestMatchesRoute(cfg, ruleIndex, pathIndex, e.Host, e.Path)
		if err != nil || !matched {
			continue
		}
		out = append(out, e)
	}
	return out
}

// FilterAccessLinesForRoute keeps raw access log lines that match the route (ri/pi).
func FilterAccessLinesForRoute(cfg *ingcore.Config, ruleIndex, pathIndex int, lines []string) []string {
	if cfg == nil {
		return nil
	}
	var out []string
	for _, line := range lines {
		e, ok := ParseAccessEntry(line)
		if !ok {
			continue
		}
		matched, err := ingcore.RequestMatchesRoute(cfg, ruleIndex, pathIndex, e.Host, e.Path)
		if err != nil || !matched {
			continue
		}
		out = append(out, line)
	}
	return out
}

// FilterWAFEventsForRoute keeps WAF events whose host/path match the route (ri/pi).
func FilterWAFEventsForRoute(cfg *ingcore.Config, ruleIndex, pathIndex int, rows []model.WAFEvent) []model.WAFEvent {
	if cfg == nil || len(rows) == 0 {
		return rows
	}
	out := make([]model.WAFEvent, 0, len(rows))
	for _, row := range rows {
		matched, err := ingcore.RequestMatchesRoute(cfg, ruleIndex, pathIndex, row.Host, row.Path)
		if err != nil || !matched {
			continue
		}
		out = append(out, row)
	}
	return out
}
