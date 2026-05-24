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

	// Host header toward upstream when service.request.host.rewrite is unset; use backend.service.mode
	// (preferred) or legacy backend.mode. Resolved via effectiveBackendMode().
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
	authTypeOAuth2 = "oauth2"

	// WWW-Authenticate challenge values
	authChallengeBasic  = "Basic realm=\"Restricted\""
	authChallengeBearer = "Bearer"

	// OAuth2 callback path and session keys
	oauth2CallbackPath           = "/oauth2/callback"
	oauth2SessionState           = "ingress_oauth2_state"
	oauth2SessionToken           = "ingress_oauth2_token"
	oauth2SessionUser            = "ingress_oauth2_user"
	oauth2SessionRedirect        = "ingress_oauth2_redirect"
)
