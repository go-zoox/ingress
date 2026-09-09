package service

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
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

// oauth2CallbackPath is the fixed path used for the OAuth2 redirect callback.
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

// ValidateOAuth2 runs the OAuth2 authentication flow.
func (s *Service) ValidateOAuth2(ctx *zoox.Context) (redirected bool, err error) {
	cfg := s.Auth.OAuth2

	// 1. Is this the callback from the identity provider?
	if ctx.Path == oauth2CallbackPath {
		return s.handleOAuth2Callback(ctx, &cfg)
	}

	// 2. Does the user already have a valid session?
	userJSON := ctx.Session().Get(sessOAuth2User)
	if userJSON != "" {
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
	saveOAuth2Flow(ctx, state, ctx.Request.URL.String())

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
		return false, fmt.Errorf("oauth2 callback: missing code or state")
	}

	// Verify state matches to prevent CSRF. Flows are keyed by state so concurrent
	// logins each keep their own original-URL redirect (no single-slot clobber).
	flowRedirect, ok := takeOAuth2Flow(ctx, state)
	if !ok {
		return false, fmt.Errorf("oauth2 callback: state mismatch")
	}

	// Create the OAuth2 client and exchange the code.
	client, clientErr := s.newOAuth2Client(ctx, cfg)
	if clientErr != nil {
		return false, fmt.Errorf("oauth2 callback: %w", clientErr)
	}

	// Following go-zoox/connect pattern: all handling (including redirect)
	// happens inside the Callback closure.
	client.Callback(code, state, func(user *gozooxoauth2.User, token *gozooxoauth2.Token, cbErr error) {
		if cbErr != nil {
			err = fmt.Errorf("oauth2 callback exchange failed: %w", cbErr)
			return
		}

		// Persist user and token in session.
		userBytes, _ := json.Marshal(user)
		ctx.Session().Set(sessOAuth2User, string(userBytes))

		tokenBytes, _ := json.Marshal(token)
		ctx.Session().Set(sessOAuth2Token, string(tokenBytes))

		// Legacy pointers are cleared here; the flow map was already consumed by
		// takeOAuth2Flow before the exchange.
		ctx.Session().Del(sessOAuth2State)
		ctx.Session().Del(sessOAuth2Redirect)

		// Redirect back to the original URL for THIS login flow.
		redirectURL := flowRedirect
		if redirectURL == "" {
			redirectURL = "/"
		}
		ctx.RedirectTemporary(redirectURL)
		redirected = true
	})
	return
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

// maxOAuth2Flows caps the number of tracked in-flight flows to keep the session
// cookie bounded against abuse (each login initiation takes one slot).
const maxOAuth2Flows = 64

func saveOAuth2Flow(ctx *zoox.Context, state, redirectURL string) {
	flows := readOAuth2Flows(ctx)
	flows[state] = redirectURL
	writeOAuth2Flows(ctx, flows)

	// Legacy single-slot pointers, retained for backward-compatible reads.
	ctx.Session().Set(sessOAuth2State, state)
	ctx.Session().Set(sessOAuth2Redirect, redirectURL)
}

func takeOAuth2Flow(ctx *zoox.Context, state string) (redirectURL string, ok bool) {
	flows := readOAuth2Flows(ctx)
	if redirectURL, ok = flows[state]; ok {
		delete(flows, state)
		writeOAuth2Flows(ctx, flows)
		return redirectURL, true
	}

	// Legacy fallback for flows started before this version.
	if ctx.Session().Get(sessOAuth2State) == state {
		redirectURL = ctx.Session().Get(sessOAuth2Redirect)
		ctx.Session().Del(sessOAuth2State)
		ctx.Session().Del(sessOAuth2Redirect)
		return redirectURL, redirectURL != ""
	}

	return "", false
}

func readOAuth2Flows(ctx *zoox.Context) map[string]string {
	raw := ctx.Session().Get(sessOAuth2Flows)
	if raw == "" {
		return map[string]string{}
	}
	var flows map[string]string
	if err := json.Unmarshal([]byte(raw), &flows); err != nil || flows == nil {
		return map[string]string{}
	}
	return flows
}

func writeOAuth2Flows(ctx *zoox.Context, flows map[string]string) {
	for len(flows) > maxOAuth2Flows {
		for k := range flows {
			delete(flows, k)
			break
		}
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
