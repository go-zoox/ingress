package rule

// BackendCache configures HTTP response caching for service, handler, and redirect backends.
// It uses the application cache (Zoox ctx.Cache()) — same Redis/memory as matcher caching when configured.
//
// Caching is off unless enabled is explicitly true.
type BackendCache struct {
	Enabled bool `config:"enabled"`
	// TTL is freshness lifetime in seconds for the stored entry when the response does not
	// provide a shorter max-age.
	TTL int64 `config:"ttl"`
	// MaxBodyBytes caps the response body size eligible for storage; larger bodies are not cached.
	MaxBodyBytes int64 `config:"max_body_bytes"`
	// KeyHash selects the fingerprint algorithm for ctx.Cache keys: "md5" (default) or "sha256".
	KeyHash string `config:"key_hash"`
	// Methods lists cacheable methods (uppercase in YAML is normalized at runtime). Default: GET, HEAD.
	Methods []string `config:"methods"`
	// KeyHeaders lists request header names included in the cache fingerprint (values are hashed).
	// Defaults are applied in core when empty.
	KeyHeaders []string `config:"key_headers"`
	// BypassRequestDirectives lists Cache-Control tokens; if any appear on the client request,
	// HTTP cache read/write is skipped and the request is handled as usual (origin/handler/redirect). Default: no-cache, no-store, max-age=0.
	BypassRequestDirectives []string `config:"bypass_request_directives"`
	// HonorPragmaNoCache treats Pragma: no-cache like Cache-Control: no-cache for bypass (default true).
	HonorPragmaNoCache *bool `config:"honor_pragma_no_cache"`
	// IgnoreResponsePrivate allows storing responses marked Cache-Control: private (default false).
	IgnoreResponsePrivate bool `config:"ignore_response_private"`
	// SkipWhenSetCookie skips storing responses that include Set-Cookie (default true).
	SkipWhenSetCookie *bool `config:"skip_when_set_cookie"`
}
