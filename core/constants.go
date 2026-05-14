package core

const (
	// backend.type YAML strings (rules[].backend.type, paths[].backend.type, fallback.type).
	//
	// backendTypeService ("service", default when inferred): reverse-proxy via backend.service only.
	//
	// backendTypeHandler ("handler"): serve via backend.handler (handler.type selects behavior).
	//
	// backendTypeRedirect ("redirect"): HTTP redirect via backend.redirect only (no upstream).
	//
	// With backend.type omitted, ingress infers from configured blocks when exactly one of service,
	// handler, or redirect applies; otherwise validation fails asking for an explicit type.
	backendTypeService  = "service"
	backendTypeHandler  = "handler"
	backendTypeRedirect = "redirect"

	// backend.mode: Host header toward upstream when service.request.host.rewrite is unset.
	backendModeInternal = "internal"
	backendModeExternal = "external"

	// Synthetic rule host when routing uses global fallback (matchHostIndex).
	fallbackRuleHost = "@@fallback"

	// host type selector
	hostTypeExact    = "exact"
	hostTypeRegex    = "regex"
	hostTypeWildcard = "wildcard"
	hostTypeAuto     = "auto"

	// handler type selector when backend.type=handler
	handlerTypeStaticResponse = "static_response"
	handlerTypeFileServer     = "file_server"
	handlerTypeTemplates      = "templates"
	handlerTypeScript         = "script"

	// script engine selector when handler.type=script
	scriptEngineJavaScript = "javascript"
	scriptEngineGo         = "go"

	// common proxy request headers
	headerXForwardedProto = "X-Forwarded-Proto"
	headerXForwardedFor   = "X-Forwarded-For"
	headerWWWAuthenticate = "WWW-Authenticate"

	// common URL schemes
	schemeHTTP  = "http"
	schemeHTTPS = "https"

	// auth type selector
	authTypeBasic  = "basic"
	authTypeBearer = "bearer"

	// WWW-Authenticate challenge values
	authChallengeBasic  = "Basic realm=\"Restricted\""
	authChallengeBearer = "Bearer"
)
