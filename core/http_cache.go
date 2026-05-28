// HTTP response caching for rules[].backend.cache (and path-level backends).
//
// Entries live in Zoox ctx.Cache() alongside matcher data; logical keys are httpCacheKeyPrefix + hash(canonical request).
// Wiring: core/build.go — redirect (before applyRedirect), handler (zooxHTTPCacheCaptureRW on GET), service upstream (OnResponse after buffering body).
package core

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"errors"
	"strings"
	"time"

	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/zoox"
)

var keyJSONPathSegmentRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Bump httpCacheKeyPrefix version suffix if the canonical serialization or semantics change.
const (
	httpCacheKeyPrefixV1 = "httpcache:v1:"
	httpCacheKeyPrefixV2 = "httpcache:v2:"

	defaultHTTPCacheKeyBodyMaxBytes = 65536
	maxHTTPCacheKeyBodyMaxBytes     = 1 << 20
	maxHTTPCacheKeyJSONPathLen      = 256
	maxHTTPCacheKeyJSONSegments     = 32

	// Canonical HTTP header names used for cache eligibility, keys, and bypass.
	headerCacheControl   = "Cache-Control"
	headerVary           = "Vary"
	headerSetCookie      = "Set-Cookie"
	headerPragma         = "Pragma"
	headerRange          = "Range"
	headerContentLength  = "Content-Length"
	headerAuthorization  = "Authorization"
	headerCookie         = "Cookie"
	headerAcceptEncoding = "Accept-Encoding"

	// Cache-Control tokens (lowercase; must match parseCacheControlDirectives normalization).
	ccTokenNoStore   = "no-store"
	ccTokenPrivate   = "private"
	ccTokenNoCache   = "no-cache"
	ccTokenMaxAge    = "max-age"
	ccTokenSMaxAge   = "s-maxage"
	ccTokenMaxAgeEq0 = "max-age=0"

	httpCacheKeyHashMD5    = "md5"
	httpCacheKeyHashSHA256 = "sha256"

	cachePathMatchAuto   = "auto"
	cachePathMatchPrefix = "prefix"
	cachePathMatchExact  = "exact"
	cachePathMatchRegex  = "regex"

	cachePathActionCache  = "cache"
	cachePathActionBypass = "bypass"

	cachePathDefaultCache  = "cache"
	cachePathDefaultBypass = "bypass"

	// Hop-by-hop header names (lowercase; lookup uses CanonicalHeaderKey then strings.ToLower).
	hopHeaderConnection         = "connection"
	hopHeaderKeepAlive          = "keep-alive"
	hopHeaderProxyConnection    = "proxy-connection"
	hopHeaderTransferEncoding   = "transfer-encoding"
	hopHeaderUpgrade            = "upgrade"
	hopHeaderTE                 = "te"
	hopHeaderTrailer            = "trailer"
	hopHeaderProxyAuthenticate  = "proxy-authenticate"
	hopHeaderProxyAuthorization = "proxy-authorization"
)

// zooxHTTPCacheCaptureRW tees response bytes to buf while still writing the real client response.
// Used for handler backends on GET so we can persist Status/Headers/Body after the handler runs.
// It embeds zoox.ResponseWriter to preserve Hijack/Flush/Status and other framework methods.
type zooxHTTPCacheCaptureRW struct {
	zoox.ResponseWriter
	buf *bytes.Buffer
}

func (w *zooxHTTPCacheCaptureRW) Write(b []byte) (int, error) {
	if w.buf != nil {
		_, _ = w.buf.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

// httpCacheEntry is the on-the-wire shape for ctx.Cache(). Redis backends JSON-marshal this; memory stores pointers.
type httpCacheEntry struct {
	StatusCode int                 `json:"status_code"`
	Header     map[string][]string `json:"header"`
	Body       []byte              `json:"body"` // empty for redirect-only entries
}

// httpCacheRuntime is rule.BackendCache normalized with defaults (only built when Enabled is true).
type httpCacheRuntime struct {
	TTL           time.Duration
	MaxBodyBytes  int64
	KeyHash       string
	MethodAllow   map[string]struct{}
	KeyHeaders    []string
	BypassTokens  map[string]struct{}
	HonorPragma   bool
	IgnorePrivate bool
	SkipSetCookie bool
	SkipVary      bool
	PathDefaultCache bool
	PathRules        []rule.BackendCachePathRuleCompiled
	// Per-request (service/handler/redirect): populated by httpCachePrepareRequest when KeyJSON is set.
	JSONKeyLines []string
	KeyJSON      []string
	KeyBodyMaxBytes int64
}

// effectiveRouteBackend returns the backend block that applies: path-level overrides host-level when present.
func effectiveRouteBackend(matchedRule *rule.Rule, pathBackend *rule.Backend) rule.Backend {
	if pathBackend != nil {
		return *pathBackend
	}
	return matchedRule.Backend
}

// normalizeHTTPCache turns YAML into runtime settings; returns nil when caching is disabled.
func normalizeHTTPCache(bc rule.BackendCache) *httpCacheRuntime {
	if !bc.Enabled {
		return nil
	}
	ttl := bc.TTL
	if ttl <= 0 {
		ttl = 300
	}
	maxBody := bc.MaxBodyBytes
	if maxBody <= 0 {
		maxBody = 2 << 20
	}
	keyHash := strings.ToLower(strings.TrimSpace(bc.KeyHash))
	if keyHash == "" {
		keyHash = httpCacheKeyHashMD5
	}

	methods := bc.Methods
	if len(methods) == 0 {
		methods = []string{http.MethodGet, http.MethodHead}
	}
	allow := make(map[string]struct{}, len(methods))
	for _, m := range methods {
		allow[strings.ToUpper(strings.TrimSpace(m))] = struct{}{}
	}

	keyHeaders := bc.KeyHeaders
	if len(keyHeaders) == 0 {
		keyHeaders = []string{headerAuthorization, headerCookie, headerAcceptEncoding}
	}
	kh := make([]string, 0, len(keyHeaders))
	for _, h := range keyHeaders {
		kh = append(kh, http.CanonicalHeaderKey(strings.TrimSpace(h)))
	}
	sort.Strings(kh)

	bypass := bc.BypassRequestDirectives
	if len(bypass) == 0 {
		bypass = []string{ccTokenNoCache, ccTokenNoStore, ccTokenMaxAgeEq0}
	}
	bt := make(map[string]struct{}, len(bypass))
	for _, b := range bypass {
		bt[strings.ToLower(strings.TrimSpace(b))] = struct{}{}
	}

	honorPragma := true
	if bc.HonorPragmaNoCache != nil {
		honorPragma = *bc.HonorPragmaNoCache
	}
	skipCookie := true
	if bc.SkipWhenSetCookie != nil {
		skipCookie = *bc.SkipWhenSetCookie
	}

	return &httpCacheRuntime{
		TTL:           time.Duration(ttl) * time.Second,
		MaxBodyBytes:  maxBody,
		KeyHash:       keyHash,
		MethodAllow:   allow,
		KeyHeaders:    kh,
		BypassTokens:  bt,
		HonorPragma:   honorPragma,
		IgnorePrivate: bc.IgnoreResponsePrivate,
		SkipSetCookie: skipCookie,
		SkipVary:      bc.SkipVary,
		PathDefaultCache: httpCachePathDefaultAllowsCache(bc.Default),
		PathRules:     append([]rule.BackendCachePathRuleCompiled(nil), bc.CompiledPathRules...),
	}
}

func httpCachePathDefaultAllowsCache(defaultAction string) bool {
	switch strings.ToLower(strings.TrimSpace(defaultAction)) {
	case cachePathDefaultBypass:
		return false
	default:
		return true
	}
}

// compileBackendCachePathRules validates and compiles backend.cache.paths for runtime matching.
func compileBackendCachePathRules(bc *rule.BackendCache) error {
	if bc == nil || len(bc.Paths) == 0 {
		bc.CompiledPathRules = nil
		return nil
	}
	out := make([]rule.BackendCachePathRuleCompiled, 0, len(bc.Paths))
	for i, pr := range bc.Paths {
		match := strings.TrimSpace(pr.Match)
		if match == "" {
			return fmt.Errorf("backend.cache.paths[%d].match must be non-empty", i)
		}
		matchType := effectiveCachePathMatchType(pr.MatchType, match)
		action := strings.ToLower(strings.TrimSpace(pr.Action))
		if action == "" {
			action = cachePathActionCache
		}
		switch action {
		case cachePathActionCache, cachePathActionBypass:
		default:
			return fmt.Errorf("backend.cache.paths[%d].action must be cache or bypass", i)
		}
		if pr.TTL < 0 {
			return fmt.Errorf("backend.cache.paths[%d].ttl must be >= 0", i)
		}
		if pr.MaxBodyBytes < 0 {
			return fmt.Errorf("backend.cache.paths[%d].max_body_bytes must be >= 0", i)
		}
		if pr.KeyBodyMaxBytes < 0 {
			return fmt.Errorf("backend.cache.paths[%d].key_body_max_bytes must be >= 0", i)
		}
		if pr.KeyBodyMaxBytes > maxHTTPCacheKeyBodyMaxBytes {
			return fmt.Errorf("backend.cache.paths[%d].key_body_max_bytes must be <= %d", i, maxHTTPCacheKeyBodyMaxBytes)
		}
		keyJSON := make([]string, 0, len(pr.KeyJSON))
		for j, p := range pr.KeyJSON {
			p = strings.TrimSpace(p)
			if p == "" {
				return fmt.Errorf("backend.cache.paths[%d].key_json[%d] must be non-empty", i, j)
			}
			if err := validateKeyJSONDotPath(p); err != nil {
				return fmt.Errorf("backend.cache.paths[%d].key_json[%d]: %w", i, j, err)
			}
			keyJSON = append(keyJSON, p)
		}
		if len(keyJSON) > 0 && action != cachePathActionCache {
			return fmt.Errorf("backend.cache.paths[%d].key_json requires action cache", i)
		}
		methods := normalizeCacheMethodsList(pr.Methods)
		if len(keyJSON) > 0 && pr.KeyBodyMaxBytes == 0 {
			pr.KeyBodyMaxBytes = defaultHTTPCacheKeyBodyMaxBytes
		}
		compiled := rule.BackendCachePathRuleCompiled{
			MatchType:       matchType,
			Cache:           action == cachePathActionCache,
			TTL:             pr.TTL,
			MaxBodyBytes:    pr.MaxBodyBytes,
			Methods:         methods,
			KeyJSON:         keyJSON,
			KeyBodyMaxBytes: pr.KeyBodyMaxBytes,
		}
		switch matchType {
		case cachePathMatchExact:
			compiled.Exact = normalizeHTTPCacheRequestPath(match)
		case cachePathMatchPrefix:
			compiled.Prefix = normalizeHTTPCacheRequestPath(match)
		case cachePathMatchRegex:
			re, err := regexp.Compile(match)
			if err != nil {
				return fmt.Errorf("backend.cache.paths[%d].match regex: %w", i, err)
			}
			compiled.Re = re
		default:
			return fmt.Errorf("backend.cache.paths[%d].match_type must be auto, prefix, exact, or regex", i)
		}
		out = append(out, compiled)
	}
	bc.CompiledPathRules = out
	return nil
}

func compileAllBackendCachePathRules(cfg *Config) error {
	for i := range cfg.Rules {
		r := &cfg.Rules[i]
		if err := compileBackendCachePathRules(&r.Backend.Cache); err != nil {
			return fmt.Errorf("rules[%d]: %w", i, err)
		}
		for j := range r.Paths {
			if err := compileBackendCachePathRules(&r.Paths[j].Backend.Cache); err != nil {
				return fmt.Errorf("rules[%d].paths[%d]: %w", i, j, err)
			}
		}
	}
	return nil
}

func effectiveCachePathMatchType(declared, match string) string {
	switch strings.ToLower(strings.TrimSpace(declared)) {
	case cachePathMatchPrefix, cachePathMatchExact, cachePathMatchRegex:
		return strings.ToLower(strings.TrimSpace(declared))
	case "", cachePathMatchAuto:
		if hostLooksLikeRegexp(match) {
			return cachePathMatchRegex
		}
		if strings.HasSuffix(match, "/") {
			return cachePathMatchPrefix
		}
		return cachePathMatchExact
	default:
		return strings.ToLower(strings.TrimSpace(declared))
	}
}

func normalizeHTTPCacheRequestPath(path string) string {
	p := strings.TrimSpace(path)
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p
}

func cachePathRuleMatches(pr rule.BackendCachePathRuleCompiled, path string) bool {
	p := normalizeHTTPCacheRequestPath(path)
	switch pr.MatchType {
	case cachePathMatchExact:
		return p == pr.Exact
	case cachePathMatchPrefix:
		return strings.HasPrefix(p, pr.Prefix)
	case cachePathMatchRegex:
		if pr.Re == nil {
			return false
		}
		return pr.Re.MatchString(p)
	default:
		return false
	}
}

type httpCachePathOverrides struct {
	TTL             time.Duration
	MaxBodyBytes    int64
	Methods         map[string]struct{} // non-nil replaces MethodAllow on effective policy
	KeyJSON         []string
	KeyBodyMaxBytes int64
}

func httpCachePathDecision(path string, pc *httpCacheRuntime) (allowCache bool, ov httpCachePathOverrides) {
	if pc == nil {
		return false, ov
	}
	ov.TTL = pc.TTL
	ov.MaxBodyBytes = pc.MaxBodyBytes
	if len(pc.PathRules) == 0 {
		return true, ov
	}
	for _, pr := range pc.PathRules {
		if !cachePathRuleMatches(pr, path) {
			continue
		}
		if !pr.Cache {
			return false, ov
		}
		if pr.TTL > 0 {
			ov.TTL = time.Duration(pr.TTL) * time.Second
		}
		if pr.MaxBodyBytes > 0 {
			ov.MaxBodyBytes = pr.MaxBodyBytes
		}
		if len(pr.Methods) > 0 {
			ov.Methods = make(map[string]struct{}, len(pr.Methods))
			for _, m := range pr.Methods {
				ov.Methods[m] = struct{}{}
			}
		}
		if len(pr.KeyJSON) > 0 {
			ov.KeyJSON = append([]string(nil), pr.KeyJSON...)
			ov.KeyBodyMaxBytes = pr.KeyBodyMaxBytes
			if ov.KeyBodyMaxBytes <= 0 {
				ov.KeyBodyMaxBytes = defaultHTTPCacheKeyBodyMaxBytes
			}
		}
		return true, ov
	}
	if pc.PathDefaultCache {
		return true, ov
	}
	return false, ov
}

// httpCachePolicyForRequest returns nil when caching is disabled or bypassed for the request path.
// Per-path TTL and max_body_bytes overrides are applied on the returned policy copy.
func httpCachePolicyForRequest(path string, pc *httpCacheRuntime) *httpCacheRuntime {
	if pc == nil {
		return nil
	}
	allow, ov := httpCachePathDecision(path, pc)
	if !allow {
		return nil
	}
	eff := *pc
	changed := false
	if ov.TTL != pc.TTL {
		eff.TTL = ov.TTL
		changed = true
	}
	if ov.MaxBodyBytes != pc.MaxBodyBytes {
		eff.MaxBodyBytes = ov.MaxBodyBytes
		changed = true
	}
	if ov.Methods != nil {
		eff.MethodAllow = ov.Methods
		changed = true
	}
	if len(ov.KeyJSON) > 0 {
		eff.KeyJSON = ov.KeyJSON
		eff.KeyBodyMaxBytes = ov.KeyBodyMaxBytes
		changed = true
	}
	if !changed {
		return pc
	}
	return &eff
}

// requestScheme prefers TLS on the connection, then X-Forwarded-Proto behind a terminating proxy.
func requestScheme(req *http.Request) string {
	if req.TLS != nil {
		return "https"
	}
	if strings.EqualFold(req.Header.Get(headerXForwardedProto), schemeHTTPS) {
		return "https"
	}
	return "http"
}

// canonicalQuerySorted rebuilds the query string with sorted keys and per-key sorted values so equivalent URLs share one cache key.
func canonicalQuerySorted(rawQuery string) string {
	if strings.TrimSpace(rawQuery) == "" {
		return ""
	}
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return rawQuery
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var pairs []string
	for _, k := range keys {
		vs := values[k]
		sort.Strings(vs)
		for _, v := range vs {
			pairs = append(pairs, url.QueryEscape(k)+"="+url.QueryEscape(v))
		}
	}
	return strings.Join(pairs, "&")
}

// headerValuesFingerprint hashes joined header values so cache keys never embed raw secrets (e.g. Cookie, Authorization).
func headerValuesFingerprint(name string, vals []string) string {
	sort.Strings(vals)
	joined := strings.Join(vals, "\n")
	sum := sha256.Sum256([]byte(joined))
	return strings.ToLower(name) + ":" + hex.EncodeToString(sum[:])
}

// buildHTTPCacheCanonical is the stable, newline-separated serialization hashed into storage keys.
// HEAD is folded to GET so HEAD can reuse a GET-cached representation without storing empty bodies as “full” responses.
func buildHTTPCacheCanonical(r *http.Request, hostname, path string, pc *httpCacheRuntime) string {
	method := strings.ToUpper(r.Method)
	if method == http.MethodHead {
		// Share cached representation with GET (same URL + vary headers).
		method = http.MethodGet
	}
	scheme := requestScheme(r)
	host := strings.ToLower(strings.TrimSpace(hostname))
	p := path
	if p == "" {
		p = "/"
	}
	query := canonicalQuerySorted(r.URL.RawQuery)

	var b strings.Builder
	fmt.Fprintf(&b, "%s\n%s\n%s\n%s\n%s\n", method, scheme, host, p, query)

	for _, name := range pc.KeyHeaders {
		vals := r.Header.Values(name)
		if len(vals) == 0 {
			fmt.Fprintf(&b, "%s:\n", name)
			continue
		}
		fmt.Fprintf(&b, "%s\n", headerValuesFingerprint(name, vals))
	}
	for _, line := range pc.JSONKeyLines {
		fmt.Fprintf(&b, "%s\n", line)
	}
	return b.String()
}

func httpCacheStorageKeyPrefix(pc *httpCacheRuntime) string {
	if pc != nil && len(pc.KeyJSON) > 0 {
		return httpCacheKeyPrefixV2
	}
	return httpCacheKeyPrefixV1
}

// buildHTTPCacheStorageKey returns the full ctx.Cache key (prefix + hex digest per pc.KeyHash).
func buildHTTPCacheStorageKey(r *http.Request, hostname, path string, pc *httpCacheRuntime) string {
	canonical := buildHTTPCacheCanonical(r, hostname, path, pc)
	var sum []byte
	switch pc.KeyHash {
	case httpCacheKeyHashSHA256:
		h := sha256.Sum256([]byte(canonical))
		sum = h[:]
	default:
		h := md5.Sum([]byte(canonical))
		sum = h[:]
	}
	return httpCacheStorageKeyPrefix(pc) + hex.EncodeToString(sum)
}

// httpCacheMethodAllowed enforces backend.cache.methods (default GET and HEAD).
func httpCacheMethodAllowed(method string, pc *httpCacheRuntime) bool {
	_, ok := pc.MethodAllow[strings.ToUpper(method)]
	return ok
}

// httpCacheShouldCaptureHandlerResponse decides whether to tee handler output for cache storage.
func httpCacheShouldCaptureHandlerResponse(method string, pc *httpCacheRuntime) bool {
	if pc == nil {
		return false
	}
	m := strings.ToUpper(method)
	if m == http.MethodGet {
		return httpCacheMethodAllowed(method, pc)
	}
	return len(pc.KeyJSON) > 0 && m == http.MethodPost && httpCacheMethodAllowed(method, pc)
}

// httpCacheRequestBypasses forces origin/handling when the client asks not to use a cached copy (or sends Range).
func httpCacheRequestBypasses(r *http.Request, pc *httpCacheRuntime) bool {
	if pc.HonorPragma {
		if strings.Contains(strings.ToLower(r.Header.Get(headerPragma)), ccTokenNoCache) {
			return true
		}
	}
	if r.Header.Get(headerRange) != "" {
		return true
	}
	cc := r.Header.Values(headerCacheControl)
	if len(cc) == 0 {
		return false
	}
	directives := parseCacheControlDirectives(strings.Join(cc, ","))
	for _, d := range directives {
		if _, ok := pc.BypassTokens[d.token]; ok {
			if d.token == ccTokenMaxAgeEq0 {
				if d.value == "0" {
					return true
				}
				continue
			}
			return true
		}
		if d.token == ccTokenMaxAge && d.value == "0" {
			if _, ok := pc.BypassTokens[ccTokenMaxAgeEq0]; ok {
				return true
			}
		}
	}
	return false
}

// ccDirective is one Cache-Control token after comma splitting (tokens and values lowercased in parseCacheControlDirectives).
type ccDirective struct {
	token string
	value string
}

// parseCacheControlDirectives splits a Cache-Control header value on commas; names/values are normalized to lower case.
func parseCacheControlDirectives(s string) []ccDirective {
	var out []ccDirective
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(strings.ToLower(part))
		if part == "" {
			continue
		}
		name, val, ok := strings.Cut(part, "=")
		name = strings.TrimSpace(name)
		val = strings.TrimSpace(val)
		if !ok {
			out = append(out, ccDirective{token: name, value: ""})
			continue
		}
		out = append(out, ccDirective{token: name, value: val})
		if name == ccTokenMaxAge && val == "0" {
			out = append(out, ccDirective{token: ccTokenMaxAgeEq0, value: "0"})
		}
	}
	return out
}

// httpCacheResponseTTL picks TTL: s-maxage or max-age if present and smaller than defaultTTL, else defaultTTL.
func httpCacheResponseTTL(res *http.Response, defaultTTL time.Duration) time.Duration {
	cc := res.Header.Values(headerCacheControl)
	if len(cc) == 0 {
		return defaultTTL
	}
	directives := parseCacheControlDirectives(strings.Join(cc, ","))
	var maxAgeSec int64 = -1
	for _, d := range directives {
		if d.token == ccTokenSMaxAge && d.value != "" {
			if n, err := strconv.ParseInt(d.value, 10, 64); err == nil && n >= 0 {
				ttl := time.Duration(n) * time.Second
				if ttl < defaultTTL {
					return ttl
				}
				return defaultTTL
			}
		}
	}
	for _, d := range directives {
		if d.token == ccTokenMaxAge && d.value != "" {
			if n, err := strconv.ParseInt(d.value, 10, 64); err == nil && n >= 0 {
				maxAgeSec = n
			}
		}
	}
	if maxAgeSec >= 0 {
		ttl := time.Duration(maxAgeSec) * time.Second
		if ttl < defaultTTL {
			return ttl
		}
		return defaultTTL
	}
	return defaultTTL
}

func httpCacheResponseNoStore(res *http.Response) bool {
	return httpCacheHeaderNoStore(strings.Join(res.Header.Values(headerCacheControl), ","))
}

func httpCacheResponsePrivate(res *http.Response) bool {
	return httpCacheHeaderPrivate(strings.Join(res.Header.Values(headerCacheControl), ","))
}

// httpCacheHeaderNoStore reports Cache-Control: no-store in a joined directive string.
func httpCacheHeaderNoStore(cc string) bool {
	for _, d := range parseCacheControlDirectives(cc) {
		if d.token == ccTokenNoStore {
			return true
		}
	}
	return false
}

// httpCacheHeaderPrivate reports Cache-Control: private in a joined directive string.
func httpCacheHeaderPrivate(cc string) bool {
	for _, d := range parseCacheControlDirectives(cc) {
		if d.token == ccTokenPrivate {
			return true
		}
	}
	return false
}

// httpCacheShouldStoreHandler applies the same storage guards as cached service responses (200 only for handlers).
func httpCacheShouldStoreHandler(status int, h http.Header, bodyLen int, pc *httpCacheRuntime) bool {
	if status != http.StatusOK {
		return false
	}
	if h.Get(headerVary) != "" && !pc.SkipVary {
		return false
	}
	cc := strings.Join(h.Values(headerCacheControl), ",")
	if httpCacheHeaderNoStore(cc) {
		return false
	}
	if httpCacheHeaderPrivate(cc) && !pc.IgnorePrivate {
		return false
	}
	if pc.SkipSetCookie && len(h.Values(headerSetCookie)) > 0 {
		return false
	}
	if int64(bodyLen) > pc.MaxBodyBytes {
		return false
	}
	return true
}

// redirectStatusFromFlags mirrors applyRedirect status codes (301/302/307/308).
func redirectStatusFromFlags(permanent, withOriginMethodAndBody bool) int {
	if withOriginMethodAndBody {
		if permanent {
			return http.StatusPermanentRedirect
		}
		return http.StatusTemporaryRedirect
	}
	if permanent {
		return http.StatusMovedPermanently
	}
	return http.StatusFound
}

// httpCacheShouldStoreRedirect allows persisting redirect rules when status and Location are cacheable (GET store path only; see build.go).
func httpCacheShouldStoreRedirect(status int, location string) bool {
	if strings.TrimSpace(location) == "" {
		return false
	}
	switch status {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther,
		http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		return true
	default:
		return false
	}
}

// httpCacheTTLFromResponseHeader applies the same max-age / s-maxage cap logic as upstream responses.
func httpCacheTTLFromResponseHeader(h http.Header, defaultTTL time.Duration) time.Duration {
	cc := h.Values(headerCacheControl)
	if len(cc) == 0 {
		return defaultTTL
	}
	return httpCacheResponseTTL(&http.Response{Header: h}, defaultTTL)
}

// httpCacheShouldStore decides whether a proxied upstream (service backend) body may be written to ctx.Cache (after full body read).
func httpCacheShouldStore(res *http.Response, bodyLen int, pc *httpCacheRuntime) bool {
	if res.StatusCode != http.StatusOK {
		return false
	}
	if res.Header.Get(headerVary) != "" && !pc.SkipVary {
		return false
	}
	if httpCacheResponseNoStore(res) {
		return false
	}
	if httpCacheResponsePrivate(res) && !pc.IgnorePrivate {
		return false
	}
	if pc.SkipSetCookie && len(res.Header.Values(headerSetCookie)) > 0 {
		return false
	}
	if int64(bodyLen) > pc.MaxBodyBytes {
		return false
	}
	return true
}

// httpCacheHopByHop lists headers we do not replay from cached entries (RFC 7230 connection-oriented headers).
var httpCacheHopByHop = map[string]struct{}{
	hopHeaderConnection:         {},
	hopHeaderKeepAlive:          {},
	hopHeaderProxyConnection:    {},
	hopHeaderTransferEncoding:   {},
	hopHeaderUpgrade:            {},
	hopHeaderTE:                 {},
	hopHeaderTrailer:            {},
	hopHeaderProxyAuthenticate:  {},
	hopHeaderProxyAuthorization: {},
}

// cloneHeadersForCache copies replay-safe headers into the JSON-friendly map stored in httpCacheEntry.
// When omitVary is true (backend.cache.skip_vary), the Vary header is dropped so hits never advertise variants.
func cloneHeadersForCache(h http.Header, omitVary bool) map[string][]string {
	out := make(map[string][]string)
	for k, vs := range h {
		kcanon := http.CanonicalHeaderKey(k)
		if omitVary && strings.EqualFold(kcanon, headerVary) {
			continue
		}
		if _, skip := httpCacheHopByHop[strings.ToLower(kcanon)]; skip {
			continue
		}
		cp := make([]string, len(vs))
		copy(cp, vs)
		out[kcanon] = cp
	}
	return out
}

// tryServeHTTPCache returns false on miss; on hit it writes the entry to the client (including redirect Location).
func tryServeHTTPCache(ctx *zoox.Context, pc *httpCacheRuntime, cacheKey string) (ok bool, status int, bodyLen int64) {
	if !httpCacheMethodAllowed(ctx.Method, pc) {
		return false, 0, 0
	}
	if httpCacheRequestBypasses(ctx.Request, pc) {
		return false, 0, 0
	}
	var entry httpCacheEntry
	if err := ctx.Cache().Get(cacheKey, &entry); err != nil {
		return false, 0, 0
	}
	writeHTTPCacheHit(ctx, &entry, pc)
	return true, entry.StatusCode, int64(len(entry.Body))
}

// writeHTTPCacheHit replays Status, headers, and body. HEAD gets the same metadata as GET but no body bytes (Content-Length from entry).
// When pc.SkipVary is true, any stored Vary header is stripped on output (safety for older entries).
func writeHTTPCacheHit(ctx *zoox.Context, entry *httpCacheEntry, pc *httpCacheRuntime) {
	h := http.Header(entry.Header)
	dst := ctx.Writer.Header()
	for k, vs := range h {
		for _, v := range vs {
			dst.Add(k, v)
		}
	}
	if pc != nil && pc.SkipVary {
		dst.Del(headerVary)
	}
	status := entry.StatusCode
	if ctx.Method == http.MethodHead {
		cl := strconv.Itoa(len(entry.Body))
		dst.Set(headerContentLength, cl)
		statusWriter(ctx, status)
		return
	}
	statusWriter(ctx, status)
	if len(entry.Body) > 0 {
		_, _ = ctx.Writer.Write(entry.Body)
	}
}

func statusWriter(ctx *zoox.Context, code int) {
	ctx.Status(code)
}

var errHTTPCacheKeyBodyTooLarge = errors.New("http cache request body too large for key_json")

func normalizeCacheMethodsList(methods []string) []string {
	if len(methods) == 0 {
		return nil
	}
	out := make([]string, 0, len(methods))
	for _, m := range methods {
		m = strings.ToUpper(strings.TrimSpace(m))
		if m != "" {
			out = append(out, m)
		}
	}
	return out
}

func methodSetIncludes(allow map[string]struct{}, method string) bool {
	if allow == nil {
		return false
	}
	_, ok := allow[strings.ToUpper(method)]
	return ok
}

func effectiveCacheMethodsForPathRule(pr rule.BackendCachePathRule, backendMethods []string) map[string]struct{} {
	methods := pr.Methods
	if len(methods) == 0 {
		methods = backendMethods
	}
	if len(methods) == 0 {
		methods = []string{http.MethodGet, http.MethodHead}
	}
	allow := make(map[string]struct{}, len(methods))
	for _, m := range methods {
		m = strings.ToUpper(strings.TrimSpace(m))
		if m != "" {
			allow[m] = struct{}{}
		}
	}
	return allow
}

func validateKeyJSONDotPath(path string) error {
	if len(path) > maxHTTPCacheKeyJSONPathLen {
		return fmt.Errorf("must be at most %d characters", maxHTTPCacheKeyJSONPathLen)
	}
	segs := strings.Split(path, ".")
	if len(segs) == 0 || len(segs) > maxHTTPCacheKeyJSONSegments {
		return fmt.Errorf("must have 1-%d dot-separated segments", maxHTTPCacheKeyJSONSegments)
	}
	for _, s := range segs {
		if !keyJSONPathSegmentRe.MatchString(s) {
			return fmt.Errorf("segment %q must match [A-Za-z_][A-Za-z0-9_]*", s)
		}
	}
	return nil
}

func requestContentTypeIsJSON(req *http.Request) bool {
	if req == nil {
		return false
	}
	ct := req.Header.Get("Content-Type")
	if ct == "" {
		return true
	}
	mediaType, _, err := mime.ParseMediaType(ct)
	if err != nil {
		return strings.Contains(strings.ToLower(ct), "application/json")
	}
	return mediaType == "application/json"
}

func buildJSONKeyCanonicalLines(body []byte, paths []string) ([]string, bool) {
	var root any
	if err := json.Unmarshal(body, &root); err != nil {
		return nil, false
	}
	top, ok := root.(map[string]any)
	if !ok {
		return nil, false
	}
	sorted := append([]string(nil), paths...)
	sort.Strings(sorted)
	lines := make([]string, 0, len(sorted))
	for _, p := range sorted {
		val, ok := jsonScalarAtDotPath(top, p)
		if !ok {
			return nil, false
		}
		lines = append(lines, fmt.Sprintf("jsonkey:%s=%s", p, val))
	}
	return lines, true
}

func jsonScalarAtDotPath(root map[string]any, dotPath string) (string, bool) {
	var cur any = root
	for _, seg := range strings.Split(dotPath, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return "", false
		}
		cur, ok = m[seg]
		if !ok {
			return "", false
		}
	}
	return formatJSONScalar(cur)
}

func formatJSONScalar(v any) (string, bool) {
	switch x := v.(type) {
	case nil:
		return "", false
	case string:
		return x, true
	case bool:
		if x {
			return "true", true
		}
		return "false", true
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64), true
	case json.Number:
		return x.String(), true
	default:
		return "", false
	}
}

func readAndReplayRequestBody(req *http.Request, limit int64) ([]byte, error) {
	if req == nil {
		return nil, nil
	}
	if req.Body == nil {
		return []byte{}, nil
	}
	defer req.Body.Close()
	lr := io.LimitReader(req.Body, limit+1)
	buf, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if int64(len(buf)) > limit {
		return nil, errHTTPCacheKeyBodyTooLarge
	}
	cloned := bytes.Clone(buf)
	req.Body = io.NopCloser(bytes.NewReader(cloned))
	req.ContentLength = int64(len(cloned))
	req.Header.Del("Transfer-Encoding")
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(cloned)), nil
	}
	return cloned, nil
}

// httpCachePrepareRequest reads JSON key fields and replays the request body for upstream proxying.
// skip=true means do not use HTTP cache for this request (still forward to origin).
func httpCachePrepareRequest(ctx *zoox.Context, pc *httpCacheRuntime) (*httpCacheRuntime, bool) {
	if pc == nil || len(pc.KeyJSON) == 0 {
		return pc, false
	}
	limit := pc.KeyBodyMaxBytes
	if limit <= 0 {
		limit = defaultHTTPCacheKeyBodyMaxBytes
	}
	if !requestContentTypeIsJSON(ctx.Request) {
		ctx.Logger.Debugf("http cache skip: content-type not json")
		return nil, true
	}
	body, err := readAndReplayRequestBody(ctx.Request, limit)
	if err != nil {
		if errors.Is(err, errHTTPCacheKeyBodyTooLarge) {
			ctx.Logger.Debugf("http cache skip: request body exceeds key_body_max_bytes")
		} else {
			ctx.Logger.Debugf("http cache skip: request body read failed: %v", err)
		}
		return nil, true
	}
	lines, ok := buildJSONKeyCanonicalLines(body, pc.KeyJSON)
	if !ok {
		ctx.Logger.Debugf("http cache skip: key_json extraction failed")
		return nil, true
	}
	eff := *pc
	eff.JSONKeyLines = lines
	return &eff, false
}
