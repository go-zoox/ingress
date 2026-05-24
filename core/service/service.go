package service

type Service struct {
	Name     string `config:"name"`
	Port     int64  `config:"port"`
	// Protocol is http or https; omit for http (config default=http).
	Protocol string `config:"protocol,default=http"`
	// Mode is internal (default) or external (upstream Host defaults to service.name). Preferred
	// over backend.mode when using backend.service; ignored when read from a non-service backend.
	Mode string `config:"mode"`
	// StripPrefix, when true on paths[].backend.service, removes the matched paths[].path prefix
	// before forwarding (expanded to request.path.rewrites at load time). Not valid on rule-level
	// or fallback backends.
	StripPrefix bool `config:"strip_prefix"`
	//
	Request  Request  `config:"request"`
	Response Response `config:"response"`
	//
	Auth Auth `config:"auth"`
	//
	HealthCheck HealthCheck `config:"health_check"`
}

type Request struct {
	Host    RequestHost       `config:"host"`
	Path    RequestPath       `config:"path"`
	Headers map[string]string `config:"headers"`
	Query   map[string]string `config:"query"`
	// Delay is the delay in milliseconds before sending the request
	Delay int64 `config:"delay"`
	// Timeout is the timeout in seconds for the request
	Timeout int64 `config:"timeout"`
}

type HealthCheck struct {
	Enable bool `config:"enable"`

	//
	Method string  `config:"method,default=GET"`
	Path   string  `config:"path,default=/health"`
	Status []int64 `config:"status,default=[200]"`

	// ok means health check is ok, ignore real check
	Ok bool `config:"ok"`
}

type RequestHost struct {
	// Rewrite, when non-nil, forces Host rewrite on or off. When nil, effective mode
	// (service.mode or backend.mode) supplies defaults.
	Rewrite *bool `config:"rewrite"`
}

type RequestPath struct {
	Rewrites []string `config:"rewrites"`
}

type Response struct {
	Headers map[string]string `config:"headers"`
}

type Auth struct {
	// Enabled controls whether authentication is enforced.
	// nil (not set in config): auth is enabled iff Type is non-empty.
	// Explicit true: auth is enabled.
	// Explicit false: auth is disabled (preserves config for later toggle).
	Enabled *bool `config:"enabled"`

	Type string `config:"type"`

	// type: basic
	Basic BasicAuth `config:"basic"`

	// type: bearer
	Bearer BearerAuth `config:"bearer"`

	// type: jwt
	Secret string `config:"secret"`

	// type: oauth2
	OAuth2 OAuth2Auth `config:"oauth2"`

	// type: oidc

	// type: service
}

type BasicAuth struct {
	Users []BasicUser `config:"users"`
}

type BasicUser struct {
	Username string `config:"username"`
	Password string `config:"password"`
}

type BearerAuth struct {
	Tokens []string `config:"tokens"`
}

type OAuth2Auth struct {
	Provider     string        `config:"provider"`
	ClientID     string        `config:"client_id"`
	ClientSecret string        `config:"client_secret"`
	RedirectURL  string        `config:"redirect_url"`
	Scopes       []string      `config:"scopes"`
	Connect      OAuth2Connect `config:"connect"`
}

type OAuth2Connect struct {
	Enabled bool            `config:"enabled"`
	JWT     OAuth2ConnectJWT `config:"jwt"`
}

type OAuth2ConnectJWT struct {
	Secret    string `config:"secret"`
	Algorithm string `config:"algorithm,default=hs256"`
	ExpiresIn string `config:"expires_in,default=5m"`
}
