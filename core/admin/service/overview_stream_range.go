package service

import (
	"strings"
	"time"
)

// metricsRangeForSubscriber resolves the query interval for an overview SSE client.
func metricsRangeForSubscriber(sub *Subscriber) MetricsRangeQuery {
	if sub == nil {
		return MetricsRangeFromWindow("5m")
	}
	fromStr := strings.TrimSpace(sub.Param("from"))
	toStr := strings.TrimSpace(sub.Param("to"))
	if fromStr != "" && toStr != "" {
		q, err := ParseMetricsRangeQuery(fromStr, toStr)
		if err != nil {
			return MetricsRangeFromWindow("5m")
		}
		now := time.Now()
		if time.Since(q.To) <= 2*time.Minute {
			q.To = now
		}
		if q.To.After(q.From) {
			return q
		}
	}
	return MetricsRangeFromWindow(normalizeMetricsWindow(subParamWindow(sub)))
}

func subscriberRangeCacheKey(sub *Subscriber) string {
	if sub == nil {
		return "window:5m"
	}
	fromStr := strings.TrimSpace(sub.Param("from"))
	toStr := strings.TrimSpace(sub.Param("to"))
	if fromStr != "" && toStr != "" {
		return "range:" + fromStr + "|" + toStr
	}
	return "window:" + normalizeMetricsWindow(subParamWindow(sub))
}
