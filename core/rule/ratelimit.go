package rule

// RateLimit configures request throttling (fixed window counter).
type RateLimit struct {
	// Enabled controls enforcement. When nil, enabled iff Requests > 0.
	Enabled *bool `config:"enabled"`
	// Requests is the maximum number of requests allowed per Period window.
	Requests int64 `config:"requests"`
	// Period is the window length in seconds.
	Period int64 `config:"period"`
	// Key selects the counter dimension: global, route, ip, or header.
	Key string `config:"key,default=ip"`
	// Header is required when Key is header (e.g. Authorization, X-API-Key).
	Header string `config:"header"`
	// TrustProxy uses X-Forwarded-For for client IP when Key is ip.
	TrustProxy bool `config:"trust_proxy"`
	// XFFIndex selects the X-Forwarded-For segment (0=leftmost; negative=from right).
	XFFIndex int `config:"xff_index"`
}
