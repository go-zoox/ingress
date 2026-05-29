package rule

// Security configures profile-based HTTP security response headers (HSTS, frame, CORS, etc.).
type Security struct {
	// Profile is strict | api | embeddable | off. Empty means off unless overridden per route.
	Profile string `config:"profile"`

	// HSTS is auto | on | off. Empty means inherit from profile (auto for strict/api/embeddable).
	HSTS string `config:"hsts"`

	// Frame is inherit | deny | sameorigin | off.
	Frame string `config:"frame"`

	// ContentTypeOptions when nil inherits from profile.
	ContentTypeOptions *bool `config:"content_type_options"`

	// ReferrerPolicy overrides profile default when non-empty; "off" disables.
	ReferrerPolicy string `config:"referrer_policy"`

	// CSP overrides profile default when non-empty; "off" disables.
	CSP string `config:"csp"`

	CORS CORS `config:"cors"`
}

// CORS configures cross-origin resource sharing for the api profile or explicit overrides.
type CORS struct {
	Enabled       *bool    `config:"enabled"`
	Origins       []string `config:"origins"`
	Methods       []string `config:"methods"`
	Headers       []string `config:"headers"`
	ExposeHeaders []string `config:"expose_headers"`
	Credentials   *bool    `config:"credentials"`
	MaxAge        int64    `config:"max_age"`
}
