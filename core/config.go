package core

type Config struct {
	Port  int64  `config:"port"`
	SSL   []SSL  `config:"ssl"`
	Rules []Rule `config:"rules"`
	//
	HTTPSPort int64 `config:"https_port"`
}

type SSL struct {
	Domain string  `config:"domain"`
	Cert   SSLCert `config:"cert"`
}

type SSLCert struct {
	Certificate    string `config:"certificate"`
	CertificateKey string `config:"certificate_key"`
}

type Rule struct {
	Host    string  `config:"host"`
	Backend Backend `config:"backend"`
	//
	Paths []Path `config:"paths"`
}

type Backend struct {
	ServiceScheme string `config:"service_scheme,default=http"`
	ServiceName   string `config:"service_name"`
	ServicePort   int64  `config:"service_port"`
	//
	Request  ConfigRequest  `config:"request"`
	Response ConfigResponse `config:"response"`
	//
	// Auth ConfigAuth `config:"auth"`
}

type Path struct {
	Path    string  `config:"path"`
	Backend Backend `config:"backend"`
}

type ConfigRequest struct {
	Rewrites []string          `config:"rewrites"`
	Headers  map[string]string `config:"headers"`
	Query    map[string]string `config:"query"`
}

type ConfigResponse struct {
	Headers map[string]string `config:"headers"`
}

// type ConfigAuth struct {
// 	Type string `config:"type"`
// 	// type == basic
// 	Username string `config:"username"`
// 	Password string `config:"password"`
// 	// type == bearer
// 	Token string `config:"token"`
// 	// type == oauth2 / oidc
// 	ClientID     string   `config:"client_id"`
// 	ClientSecret string   `config:"client_secret"`
// 	AllowIds     []string `config:"allow_ids"`
// }
