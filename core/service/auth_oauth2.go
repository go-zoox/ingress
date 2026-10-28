package service

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-zoox/jwt"
	gozooxoauth2 "github.com/go-zoox/oauth2"
	"github.com/go-zoox/oauth2/create"
	"github.com/go-zoox/zoox"
)

// ---------------------------------------------------------------------------
// provider default URL mapping — used when the user only specifies a provider
// name and the corresponding go-zoox/oauth2 provider package handles the rest.
// ---------------------------------------------------------------------------

var supportedOAuth2Providers = map[string]bool{
	"doreamon":  true,
	"github":    true,
	"feishu":    true,
	"gitlab":    true,
	"slack":     true,
	"kakao":     true,
	"google":    true,
	"microsoft": true,
	"auth0":     true,
	"okta":      true,
}

// defaultOAuth2Scopes provides sensible default scopes for each provider when
// the user does not explicitly configure scopes in the YAML config.
var defaultOAuth2Scopes = map[string][]string{
	"github":    {"user:email"},
	"gitlab":    {"read_user"},
	"google":    {"openid", "profile", "email"},
	"microsoft": {"openid", "profile", "email"},
	"feishu":    {"user:email"},
	"slack":     {"users:read"},
	"auth0":     {"openid", "profile", "email"},
	"okta":      {"openid", "profile", "email"},
}

// ---------------------------------------------------------------------------
// ValidateOAuth2 handles the OAuth2 authentication flow within a zoox.Context.
//
// It returns (redirected, error). When redirected is true, the caller MUST
// stop further processing (the response has already been written).
// When redirected is false and error is nil, the user is authenticated;
// the caller should continue with the normal proxy flow.
// ---------------------------------------------------------------------------

// oauth2CallbackPath is the default path used for the OAuth2 redirect callback.
// When auth.oauth2.redirect_url is set, its path is used instead (see
// oauth2CallbackPathFor) so a provider-registered callback URL such as
// /login/doreamon/callback keeps working.
const oauth2CallbackPath = "/oauth2/callback"

// session key constants used with zoox ctx.Session().
const (
	sessOAuth2State    = "ingress_oauth2_state"
	sessOAuth2Token    = "ingress_oauth2_token"
	sessOAuth2User     = "ingress_oauth2_user"
	sessOAuth2Redirect = "ingress_oauth2_redirect"
	// sessOAuth2Flows holds a JSON map {state: originalURL} of in-flight logins so
	// concurrent flows on one session don't clobber a single state/redirect slot.
	sessOAuth2Flows = "ingress_oauth2_flows"
)

// Stale-state recovery is bounded to one automatic attempt. Two markers are used
// so the bound does not depend on a single mechanism: the retry URL carries a
// query parameter that is encoded into the next CSRF state (survives even when
// cookies are dropped), and a short-lived cookie covers redirect chains that would
// otherwise lose the query. Either marker present -> plain 401, no second retry.
const (
	oauth2RetryQueryParam  = "_ingress_oauth2_retry"
	oauth2RetryStatePrefix = "r1."
	oauth2RetryCookie      = "ingress_oauth2_retry"
)

// Session cookie budget.
//
// The OAuth2 session (CSRF state, post-login URL, user, token) lives in an
// encrypted *client-side* cookie (zoox session, default cookie "gsession").
// Browsers silently drop cookies whose name+value exceeds ~4096 bytes; from
// then on every response that tries to persist the session is ignored, so a
// new CSRF state can never be stored and the callback fails with 401 until the
// cookie is invalidated (e.g. restarting ingress rotates the per-process
// session key). Keep the OAuth2 payload small and bounded so that cannot
// happen: the flow map used to grow to 64 entries and blow the limit.
const (
	// maxOAuth2Flows caps how many in-flight logins are tracked per browser.
	maxOAuth2Flows = 8
	// maxOAuth2FlowRedirectURLBytes caps the stored post-login URL
	// (query/fragment are dropped first when a URL is too long).
	maxOAuth2FlowRedirectURLBytes = 200
	// maxOAuth2FlowsBytes caps the JSON payload of the in-flight flow map.
	maxOAuth2FlowsBytes = 1536
	// maxOAuth2UserJSONBytes caps the JSON payload of the authenticated user.
	maxOAuth2UserJSONBytes = 768
)

// ValidateOAuth2 runs the OAuth2 authentication flow.
func (s *Service) ValidateOAuth2(ctx *zoox.Context) (redirected bool, err error) {
	cfg := s.Auth.OAuth2

	// 1. Is this the callback from the identity provider?
	if isOAuth2CallbackPath(ctx.Path, &cfg) {
		return s.handleOAuth2Callback(ctx, &cfg)
	}

	// 2. Does the user already have a valid session?
	if ctx.Session().Get(sessOAuth2User) != "" {
		// Session exists — user is authenticated.
		return false, nil
	}

	// 3. No session — redirect to the identity provider.
	client, err := s.newOAuth2Client(ctx, &cfg)
	if err != nil {
		return false, fmt.Errorf("failed to create oauth2 client: %w", err)
	}

	state, err := generateRandomState()
	if err != nil {
		return false, fmt.Errorf("failed to generate oauth2 state: %w", err)
	}

	// Save the original request URL so we can redirect back after login.
	// Stored per-login (keyed by state) so concurrent flows don't clobber each other.
	originalURL := ctx.Request.URL.String()
	if ctx.Query().Get(oauth2RetryQueryParam).String() == "1" {
		// This login is the automatic retry issued by stale-state recovery: mark the
		// state so a second consecutive failure is bounded, and do not redirect back
		// to the marker URL afterwards.
		state = oauth2RetryStatePrefix + state
		originalURL = "/"
	}
	saveOAuth2Flow(ctx, state, originalURL)

	client.Authorize(state, func(loginURL string) {
		ctx.RedirectTemporary(loginURL)
	})
	return true, nil
}

// handleOAuth2Callback exchanges the authorization code for a token,
// fetches user info, stores it in the session, and redirects back to
// the original protected URL.
// All business logic lives inside the Callback closure, following the
// go-zoox/oauth2 library's intended pattern (as used in go-zoox/connect).
func (s *Service) handleOAuth2Callback(ctx *zoox.Context, cfg *OAuth2Auth) (redirected bool, err error) {
	code := ctx.Query().Get("code").String()
	state := ctx.Query().Get("state").String()

	if code == "" || state == "" {
		// Providers report errors (access_denied, invalid scope, …) on the
		// callback as ?error=…&error_description=… with no code. Surface that
		// instead of a generic "missing code or state".
		if providerErr := ctx.Query().Get("error").String(); providerErr != "" {
			return false, fmt.Errorf("oauth2 callback: identity provider returned %q: %s",
				providerErr, ctx.Query().Get("error_description").String())
		}
		return false, fmt.Errorf("oauth2 callback: missing code or state")
	}

	// Verify state matches to prevent CSRF. Flows are keyed by state so concurrent
	// logins each keep their own original-URL redirect (no single-slot clobber).
	//
	// The flow is only *read* here; it is consumed after a successful exchange so
	// a transient provider/network failure can be retried by reloading the
	// callback URL instead of permanently turning into "state mismatch".
	flowRedirect, ok := lookupOAuth2Flow(ctx, state)
	if !ok {
		return s.recoverOAuth2StaleState(ctx, state)
	}

	// Create the OAuth2 client and exchange the code.
	client, clientErr := s.newOAuth2Client(ctx, cfg)
	if clientErr != nil {
		return false, fmt.Errorf("oauth2 callback: %w", clientErr)
	}

	// go-zoox/oauth2 dereferences a nil error when the provider answers
	// `/token` with HTTP 200 and a JSON body carrying a non-zero `code`; turn such
	// a panic into a regular authentication failure (401) instead of a 500.
	defer func() {
		if r := recover(); r != nil {
			redirected = false
			err = fmt.Errorf("oauth2 callback exchange panic: %v", r)
		}
	}()

	// Following go-zoox/connect pattern: all handling (including redirect)
	// happens inside the Callback closure.
	client.Callback(code, state, func(user *gozooxoauth2.User, _ *gozooxoauth2.Token, cbErr error) {
		if cbErr != nil {
			err = fmt.Errorf("oauth2 callback exchange failed: %w", cbErr)
			return
		}

		// Redirect back to the original URL for THIS login flow.
		ctx.RedirectTemporary(completeOAuth2Login(ctx, state, flowRedirect, user))
		redirected = true
	})
	return
}

// completeOAuth2Login finishes a successful login: it consumes the in-flight
// flow, persists the (trimmed) user profile and returns the post-login redirect.
//
// The access/refresh token is intentionally NOT stored: nothing reads it, and
// every byte packed into the client-side session cookie brings the ~4KB browser
// cookie limit closer (the historic cause of callbacks failing with 401).
func completeOAuth2Login(ctx *zoox.Context, state, flowRedirect string, user *gozooxoauth2.User) (redirectURL string) {
	// Serialize first: nothing below may fail after the flow is consumed.
	userJSON := marshalOAuth2SessionUser(user)

	takeOAuth2Flow(ctx, state)

	ctx.Session().Set(sessOAuth2User, userJSON)
	if ctx.Session().Get(sessOAuth2Token) != "" {
		ctx.Session().Del(sessOAuth2Token)
	}
	if hasOAuth2RetryCookie(ctx) {
		clearOAuth2RetryCookie(ctx)
	}

	if flowRedirect == "" {
		return "/"
	}
	return flowRedirect
}

// recoverOAuth2StaleState handles a callback whose CSRF state is not present in
// the browser's session cookie. That happens when:
//   - the provider sends a stale/replayed callback URL,
//   - the cookie was issued by another process/replica (session key rotated or
//     replicas without a shared secret_key),
//   - the browser silently dropped the cookie update because the session cookie
//     had grown past the ~4KB cookie limit (the historic "restart ingress to
//     recover" case).
//
// Instead of answering 401 and requiring an ingress restart, delete the unusable
// session cookie (a single tiny Set-Cookie the browser accepts even when it had
// been rejecting the previous oversized value) and restart the login flow once.
// Both the retry URL/state marker and a short-lived cookie bound this to one
// automatic attempt.
func (s *Service) recoverOAuth2StaleState(ctx *zoox.Context, state string) (redirected bool, err error) {
	// Already authenticated: a stale or replayed callback is harmless.
	if ctx.Session().Get(sessOAuth2User) != "" {
		ctx.RedirectTemporary("/")
		return true, nil
	}

	// Drop the whole (gateway) session cookie before retrying. Deleting the cookie
	// is the only way to guarantee a single small Set-Cookie header: the session
	// package flushes the entire payload on every mutation, so rewriting a trimmed
	// payload would first emit the legacy 4KB+ blob again (which the browser — and
	// some reverse proxies — reject). Safe here: a gateway session holds OAuth2
	// login state only and the user is not authenticated on this branch.
	deleteOAuth2SessionCookie(ctx)

	if strings.HasPrefix(state, oauth2RetryStatePrefix) || hasOAuth2RetryCookie(ctx) {
		// The automatic retry failed too: stop instead of looping between
		// /authorize and /callback. The stale cookie is already gone, so the next
		// manual attempt starts clean.
		clearOAuth2RetryCookie(ctx)
		return false, fmt.Errorf("oauth2 callback: state mismatch (CSRF state not found in session cookie; " +
			"the session cookie may have been too large or issued by another replica; " +
			"clear the site cookies or check that every replica shares the same secret_key)")
	}

	setOAuth2RetryCookie(ctx)
	ctx.RedirectTemporary("/?" + oauth2RetryQueryParam + "=1")
	return true, nil
}

// oauth2RetryCookie helpers: short-lived marker (60s) used to bound stale-state
// recovery across redirect chains that drop the query parameter.
func setOAuth2RetryCookie(ctx *zoox.Context) {
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     oauth2RetryCookie,
		Value:    "1",
		Path:     "/",
		MaxAge:   60,
		HttpOnly: true,
		Secure:   requestScheme(ctx.Request) == "https",
		SameSite: http.SameSiteLaxMode,
	})
}

func hasOAuth2RetryCookie(ctx *zoox.Context) bool {
	c, err := ctx.Request.Cookie(oauth2RetryCookie)
	return err == nil && c.Value != ""
}

func clearOAuth2RetryCookie(ctx *zoox.Context) {
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     oauth2RetryCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   requestScheme(ctx.Request) == "https",
		SameSite: http.SameSiteLaxMode,
	})
}

// ---------------------------------------------------------------------------
// OAuth2 flow tracking
//
// The CSRF state and the post-login redirect URL are stored per flow (keyed by
// state) so concurrent logins on the same browser/session do not clobber a
// single shared slot — a common cause of intermittent 401 on the callback.
// A legacy single-state fallback keeps in-flight flows created by older versions
// working across a rolling upgrade.
// ---------------------------------------------------------------------------

func saveOAuth2Flow(ctx *zoox.Context, state, redirectURL string) {
	flows := readOAuth2Flows(ctx)
	flows = append(flows, oauth2Flow{State: state, Redirect: clampOAuth2RedirectURL(redirectURL)})
	writeOAuth2Flows(ctx, pruneOAuth2Flows(flows))

	// Legacy single-slot pointers are no longer written (they duplicated the
	// state/redirect in the cookie); drop them so existing oversized session
	// cookies shrink on the next login.
	if ctx.Session().Get(sessOAuth2State) != "" {
		ctx.Session().Del(sessOAuth2State)
	}
	if ctx.Session().Get(sessOAuth2Redirect) != "" {
		ctx.Session().Del(sessOAuth2Redirect)
	}
}

// lookupOAuth2Flow returns the post-login redirect URL for an in-flight state
// without consuming it.
func lookupOAuth2Flow(ctx *zoox.Context, state string) (redirectURL string, ok bool) {
	for _, flow := range readOAuth2Flows(ctx) {
		if flow.State == state {
			return flow.Redirect, true
		}
	}

	// Legacy fallback for flows started before this version.
	if ctx.Session().Get(sessOAuth2State) == state {
		return ctx.Session().Get(sessOAuth2Redirect), true
	}

	return "", false
}

// takeOAuth2Flow consumes an in-flight flow (called after a successful login).
func takeOAuth2Flow(ctx *zoox.Context, state string) (redirectURL string, ok bool) {
	redirectURL, ok = lookupOAuth2Flow(ctx, state)
	if !ok {
		return "", false
	}

	flows := readOAuth2Flows(ctx)
	remaining := flows[:0]
	for _, flow := range flows {
		if flow.State != state {
			remaining = append(remaining, flow)
		}
	}
	writeOAuth2Flows(ctx, remaining)

	if ctx.Session().Get(sessOAuth2State) == state {
		ctx.Session().Del(sessOAuth2State)
		ctx.Session().Del(sessOAuth2Redirect)
	}

	return redirectURL, true
}

// clampOAuth2RedirectURL bounds the URL stored in the session cookie. Over-long
// URLs keep their path (so the user still lands on the right page) but drop the
// query/fragment; when the path itself is too long the redirect falls back to "/".
func clampOAuth2RedirectURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "/"
	}
	if len(raw) <= maxOAuth2FlowRedirectURLBytes {
		return raw
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "/"
	}
	u.RawQuery = ""
	u.Fragment = ""
	if trimmed := u.String(); len(trimmed) <= maxOAuth2FlowRedirectURLBytes {
		return trimmed
	}
	return "/"
}

// pruneOAuth2Flows bounds the flow list by both entry count and JSON size,
// evicting the oldest entries first and always keeping the newest (the one just
// stored). The session is a client-side cookie and browsers reject values over
// ~4KB, which would silently break every callback.
func pruneOAuth2Flows(flows []oauth2Flow) []oauth2Flow {
	for len(flows) > 1 && (len(flows) > maxOAuth2Flows || oauth2FlowsSize(flows) > maxOAuth2FlowsBytes) {
		flows = flows[1:]
	}
	return flows
}

func oauth2FlowsSize(flows []oauth2Flow) int {
	b, err := json.Marshal(flows)
	if err != nil {
		return 0
	}
	return len(b)
}

// oauth2Flow is one in-flight login: the CSRF state and the URL to return to.
// It is serialized as [state, redirect] so the list keeps insertion order
// (oldest first), which lets pruning evict the oldest flow deterministically.
type oauth2Flow struct {
	State    string
	Redirect string
}

func (f oauth2Flow) MarshalJSON() ([]byte, error) {
	return json.Marshal([2]string{f.State, f.Redirect})
}

func (f *oauth2Flow) UnmarshalJSON(b []byte) error {
	var pair [2]string
	if err := json.Unmarshal(b, &pair); err != nil {
		return err
	}
	f.State, f.Redirect = pair[0], pair[1]
	return nil
}

// oauth2SessionUser is the subset of the provider profile persisted in the
// client-side session cookie. It covers what BuildConnectJWT injects and
// deliberately drops permissions/groups arrays to keep the cookie small.
type oauth2SessionUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

// marshalOAuth2SessionUser serializes the user profile for the session cookie,
// falling back to a minimal identity when the profile is unexpectedly large.
func marshalOAuth2SessionUser(user *gozooxoauth2.User) string {
	if user == nil {
		return ""
	}

	su := oauth2SessionUser{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Nickname: user.Nickname,
		Avatar:   user.Avatar,
	}

	b, err := json.Marshal(su)
	if err != nil {
		return ""
	}
	if len(b) > maxOAuth2UserJSONBytes {
		// Drop the (largest, non-essential) presentation fields.
		b, err = json.Marshal(oauth2SessionUser{ID: su.ID, Username: su.Username, Email: su.Email})
		if err != nil {
			return ""
		}
	}
	return string(b)
}

// deleteOAuth2SessionCookie deletes the gateway session cookie with a single tiny
// Set-Cookie header. This is what makes an oversized session self-heal without an
// ingress restart: the browser accepts the deletion (unlike the oversized value
// it had been rejecting), so the next login can persist its CSRF state again.
func deleteOAuth2SessionCookie(ctx *zoox.Context) {
	name := strings.TrimSpace(ctx.App.Config.Session.Key)
	if name == "" {
		name = "gsession"
	}
	path := strings.TrimSpace(ctx.App.Config.Session.Path)
	if path == "" {
		path = "/"
	}
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     path,
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// isOAuth2CallbackPath reports whether the request path is the OAuth2 callback.
// The default callback path is always accepted (so in-flight logins survive a
// redirect_url change); a custom redirect_url path is accepted as well.
func isOAuth2CallbackPath(path string, cfg *OAuth2Auth) bool {
	if sameOAuth2CallbackPath(path, oauth2CallbackPath) {
		return true
	}
	return sameOAuth2CallbackPath(path, oauth2CallbackPathFor(cfg))
}

// sameOAuth2CallbackPath compares two callback paths, tolerating a trailing slash
// (providers/proxies sometimes normalise it).
func sameOAuth2CallbackPath(a, b string) bool {
	if a == b {
		return true
	}
	return strings.TrimRight(a, "/") == strings.TrimRight(b, "/")
}

// oauth2CallbackPathFor derives the callback path from the configured
// redirect_url. It returns the default path when redirect_url is empty or has no
// usable path.
func oauth2CallbackPathFor(cfg *OAuth2Auth) string {
	raw := strings.TrimSpace(cfg.RedirectURL)
	if raw == "" {
		return oauth2CallbackPath
	}

	u, err := url.Parse(raw)
	if err != nil {
		return oauth2CallbackPath
	}
	path := strings.TrimSpace(u.Path)
	if path == "" || path == "/" {
		return oauth2CallbackPath
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func readOAuth2Flows(ctx *zoox.Context) []oauth2Flow {
	raw := ctx.Session().Get(sessOAuth2Flows)
	if raw == "" {
		return nil
	}

	var flows []oauth2Flow
	if err := json.Unmarshal([]byte(raw), &flows); err == nil {
		return flows
	}

	// Legacy format: a JSON map {state: redirect} written by older versions.
	var legacy map[string]string
	if err := json.Unmarshal([]byte(raw), &legacy); err != nil || legacy == nil {
		return nil
	}
	for state, redirect := range legacy {
		flows = append(flows, oauth2Flow{State: state, Redirect: redirect})
	}
	return flows
}

func writeOAuth2Flows(ctx *zoox.Context, flows []oauth2Flow) {
	if len(flows) == 0 {
		ctx.Session().Del(sessOAuth2Flows)
		return
	}
	b, err := json.Marshal(flows)
	if err != nil {
		ctx.Session().Del(sessOAuth2Flows)
		return
	}
	ctx.Session().Set(sessOAuth2Flows, string(b))
}

// newOAuth2Client creates a go-zoox/oauth2 client from the ingress config.
func (s *Service) newOAuth2Client(ctx *zoox.Context, cfg *OAuth2Auth) (gozooxoauth2.Client, error) {
	if !supportedOAuth2Providers[cfg.Provider] {
		return nil, fmt.Errorf("unsupported oauth2 provider: %s", cfg.Provider)
	}

	redirectURL := cfg.RedirectURL
	if redirectURL == "" {
		// Auto-generate redirect URL from the current request. Honor a TLS-terminating
		// reverse proxy via X-Forwarded-Proto (same heuristic as HTTP->HTTPS redirects),
		// so multi-replica deployments behind an LB register the correct https callback.
		scheme := requestScheme(ctx.Request)
		redirectURL = fmt.Sprintf("%s://%s%s", scheme, ctx.Host(), oauth2CallbackPath)
	}

	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = defaultOAuth2Scopes[cfg.Provider]
	}
	scope := strings.Join(scopes, " ")
	oauth2Cfg := &gozooxoauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURI:  redirectURL,
		Scope:        scope,
	}

	return create.Create(cfg.Provider, oauth2Cfg)
}

// requestScheme reports whether the inbound request is HTTPS, honoring a
// TLS-terminating reverse proxy via X-Forwarded-Proto (same heuristic as the
// HTTP->HTTPS redirect logic elsewhere in ingress).
func requestScheme(req *http.Request) string {
	if req.TLS != nil {
		return "https"
	}
	if strings.EqualFold(strings.TrimSpace(req.Header.Get("X-Forwarded-Proto")), "https") {
		return "https"
	}
	return "http"
}

// ---------------------------------------------------------------------------
// Connect headers (JWT injection) — used when connect.enabled is true.
// ---------------------------------------------------------------------------

// BuildConnectJWT signs the OAuth2 user into an X-Connect-Token JWT.
func (s *Service) BuildConnectJWT(user *gozooxoauth2.User) (string, error) {
	jwtCfg := s.Auth.OAuth2.Connect.JWT
	alg := strings.ToUpper(jwtCfg.Algorithm)
	if alg == "" {
		alg = "HS256"
	}

	expiresIn := jwtCfg.ExpiresIn
	if expiresIn == "" {
		expiresIn = "5m"
	}
	dur, err := time.ParseDuration(expiresIn)
	if err != nil {
		return "", fmt.Errorf("invalid jwt expires_in: %w", err)
	}

	now := time.Now()
	j := jwt.New(jwtCfg.Secret, &jwt.Options{
		Algorithm: alg,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(dur).Unix(),
	})

	token, err := j.Sign(map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"nickname": user.Nickname,
		"avatar":   user.Avatar,
	})
	if err != nil {
		return "", fmt.Errorf("failed to sign connect jwt: %w", err)
	}

	return token, nil
}

// InjectConnectHeaders reads the OAuth2 user from the session, builds a JWT,
// and injects X-Connect-Token and X-Connect-Timestamp into the incoming request
// headers so the upstream service can identify the authenticated user.
func (s *Service) InjectConnectHeaders(ctx *zoox.Context) error {
	userJSON := ctx.Session().Get(sessOAuth2User)
	if userJSON == "" {
		return fmt.Errorf("no oauth2 user in session")
	}

	var user gozooxoauth2.User
	if err := json.Unmarshal([]byte(userJSON), &user); err != nil {
		return fmt.Errorf("failed to parse oauth2 user from session: %w", err)
	}

	token, err := s.BuildConnectJWT(&user)
	if err != nil {
		return err
	}

	ctx.Request.Header.Set("X-Connect-Token", token)
	ctx.Request.Header.Set("X-Connect-Timestamp", fmt.Sprintf("%d", time.Now().UnixMilli()))

	return nil
}

// generateRandomState creates a cryptographically random string for CSRF protection.
func generateRandomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
