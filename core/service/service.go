package service

type Service struct {
	Name     string `config:"name"`
	Port     int64  `config:"port"`
	Protocol string `config:"protocol,default=http"`
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
	Rewrite bool `config:"rewrite"`
}

type RequestPath struct {
	Rewrites []string `config:"rewrites"`
}

type Response struct {
	Headers map[string]string `config:"headers"`
}

type Auth struct {
	Type string `config:"type"`

	// type: basic
	Basic BasicAuth `config:"basic"`

	// type: bearer
	Bearer BearerAuth `config:"bearer"`

	// type: jwt
	Secret string `config:"secret"`

	// type: oauth2
	Provider     string   `config:"provider"`
	ClientID     string   `config:"client_id"`
	ClientSecret string   `config:"client_secret"`
	RedirectURL  string   `config:"redirect_url"`
	Scopes       []string `config:"scopes"`

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
