package core

const (
	// backend type selector
	backendTypeService = "service"
	backendTypeHandler = "handler"

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
