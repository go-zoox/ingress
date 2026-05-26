package core

// WAFCallback is called by core when WAF blocks or audits a request.
// The admin package implements this to persist events to DB.
type WAFCallback interface {
	OnWAFEvent(action, rule, host, path, clientIP, userAgent string)
}
