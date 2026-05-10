package waf

// Pattern and target names for waf.rules (YAML type / targets fields).
const (
	PatternTypeRegex    = "regex"
	PatternTypeContains = "contains"

	TargetPath    = "path"
	TargetQuery   = "query"
	TargetURI     = "uri"
	TargetHeaders = "headers"
	headerPrefix  = "header:"
)

const headerXForwardedFor = "X-Forwarded-For"
