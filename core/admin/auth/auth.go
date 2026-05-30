package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	admincfg "github.com/go-zoox/ingress/core/admin/config"
	"github.com/go-zoox/ingress/core/admin/service/rbac"
	gozooxoauth2 "github.com/go-zoox/oauth2"
	"github.com/go-zoox/oauth2/create"
	"github.com/go-zoox/zoox"
	"golang.org/x/crypto/bcrypt"
)

const (
	ContextUserKey        = "admin.auth.user"
	ContextDisplayNameKey = "admin.auth.display_name"

	sessUser           = "ingress_admin_user"
	sessDisplayName    = "ingress_admin_display_name"
	sessOAuth2State    = "ingress_admin_oauth2_state"
	sessOAuth2Redirect = "ingress_admin_oauth2_redirect"
	sessOAuth2User     = "ingress_admin_oauth2_user"

	oauthCallbackPath = "/api/v1/auth/oauth/callback"
)

var (
	// ErrNoConsoleAccess means credentials are valid but the user has no visible menus.
	ErrNoConsoleAccess = errors.New("no permission to access admin console")
)

var supportedOAuthProviders = map[string]bool{
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

var defaultOAuthScopes = map[string][]string{
	"github":    {"user:email"},
	"gitlab":    {"read_user"},
	"google":    {"openid", "profile", "email"},
	"microsoft": {"openid", "profile", "email"},
	"feishu":    {"user:email"},
	"slack":     {"users:read"},
	"auth0":     {"openid", "profile", "email"},
	"okta":      {"openid", "profile", "email"},
}

// UserInfo is the authenticated Admin Console operator.
type UserInfo struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

// ConfigView is returned to the frontend login/bootstrap flow.
type ConfigView struct {
	Type            string    `json:"type"`
	Authenticated   bool      `json:"authenticated"`
	User            *UserInfo `json:"user,omitempty"`
	OAuthLoginURL   string    `json:"oauth_login_url,omitempty"`
}

// Service implements Admin Console authentication.
type Service struct {
	cfg  *admincfg.Config
	rbac *rbac.Service
}

func New(cfg *admincfg.Config, rbacSvc *rbac.Service) *Service {
	return &Service{cfg: cfg, rbac: rbacSvc}
}

func (s *Service) Type() string {
	if s == nil || s.cfg == nil {
		return "basic"
	}
	return admincfg.EffectiveAuthType(s.cfg.Auth.Type)
}

func (s *Service) RequiresAuth() bool {
	return s.Type() != "none"
}

func (s *Service) ConfigView(ctx *zoox.Context) ConfigView {
	out := ConfigView{Type: s.Type()}
	user := s.CurrentUser(ctx)
	if user != nil {
		out.Authenticated = true
		out.User = user
	}
	if s.Type() == "oauth" {
		out.OAuthLoginURL = "/api/v1/auth/oauth/login"
	}
	return out
}

func (s *Service) CurrentUser(ctx *zoox.Context) *UserInfo {
	if ctx == nil {
		return nil
	}
	username := strings.TrimSpace(ctx.Get(ContextUserKey))
	if username == "" {
		username = strings.TrimSpace(ctx.Session().Get(sessUser))
	}
	if username == "" {
		return nil
	}
	displayName := strings.TrimSpace(ctx.Get(ContextDisplayNameKey))
	if displayName == "" {
		displayName = strings.TrimSpace(ctx.Session().Get(sessDisplayName))
	}
	if displayName == "" {
		displayName = username
	}
	return &UserInfo{Username: username, DisplayName: displayName}
}

func (s *Service) UsernameFromContext(ctx *zoox.Context) string {
	user := s.CurrentUser(ctx)
	if user == nil {
		return ""
	}
	return user.Username
}

func (s *Service) IsPublicPath(method, path string) bool {
	if !strings.HasPrefix(path, "/api/v1/") {
		return true
	}
	switch path {
	case "/api/v1/auth/config",
		"/api/v1/auth/login",
		"/api/v1/auth/logout",
		"/api/v1/auth/oauth/login",
		"/api/v1/auth/oauth/callback":
		return true
	default:
		return false
	}
}

func (s *Service) Middleware() zoox.HandlerFunc {
	return func(ctx *zoox.Context) {
		if !strings.HasPrefix(ctx.Path, "/api/v1/") {
			ctx.Next()
			return
		}
		if s.IsPublicPath(ctx.Method, ctx.Path) {
			ctx.Next()
			return
		}
		if !s.RequiresAuth() {
			ctx.Next()
			return
		}
		user := s.loadSessionUser(ctx)
		if user == nil {
			fail(ctx, http.StatusUnauthorized, "authentication required")
			return
		}
		ctx.Set(ContextUserKey, user.Username)
		ctx.Set(ContextDisplayNameKey, user.DisplayName)
		ctx.Next()
	}
}

func (s *Service) loadSessionUser(ctx *zoox.Context) *UserInfo {
	username := strings.TrimSpace(ctx.Session().Get(sessUser))
	if username == "" {
		return nil
	}
	displayName := strings.TrimSpace(ctx.Session().Get(sessDisplayName))
	if displayName == "" {
		displayName = username
	}
	return &UserInfo{Username: username, DisplayName: displayName}
}

func (s *Service) setSessionUser(ctx *zoox.Context, user *UserInfo) {
	if user == nil {
		return
	}
	ctx.Session().Set(sessUser, user.Username)
	ctx.Session().Set(sessDisplayName, user.DisplayName)
	ctx.Set(ContextUserKey, user.Username)
	ctx.Set(ContextDisplayNameKey, user.DisplayName)
}

func (s *Service) clearSession(ctx *zoox.Context) {
	ctx.Session().Del(sessUser)
	ctx.Session().Del(sessDisplayName)
	ctx.Session().Del(sessOAuth2State)
	ctx.Session().Del(sessOAuth2Redirect)
	ctx.Session().Del(sessOAuth2User)
}

func (s *Service) LoginBasic(ctx *zoox.Context, username, password string) (*UserInfo, error) {
	if s.Type() != "basic" {
		return nil, fmt.Errorf("basic login is disabled")
	}
	row, err := s.rbac.Authenticate(username, password)
	if err != nil {
		return nil, err
	}
	if err := s.ensureConsoleAccess(row.Username); err != nil {
		return nil, err
	}
	user := &UserInfo{Username: row.Username, DisplayName: row.DisplayName}
	s.setSessionUser(ctx, user)
	return user, nil
}

func (s *Service) ensureConsoleAccess(username string) error {
	nav, err := s.rbac.ListNavigation(username)
	if err != nil {
		return err
	}
	if len(nav.Groups) == 0 {
		return ErrNoConsoleAccess
	}
	return nil
}

func (s *Service) Logout(ctx *zoox.Context) {
	s.clearSession(ctx)
}

func (s *Service) StartOAuth(ctx *zoox.Context, redirect string) (bool, error) {
	if s.Type() != "oauth" {
		return false, fmt.Errorf("oauth login is disabled")
	}
	if strings.TrimSpace(redirect) == "" {
		redirect = "/"
	}
	if !strings.HasPrefix(redirect, "/") {
		redirect = "/"
	}

	client, err := s.newOAuthClient(ctx)
	if err != nil {
		return false, err
	}
	state, err := randomState()
	if err != nil {
		return false, err
	}
	ctx.Session().Set(sessOAuth2State, state)
	ctx.Session().Set(sessOAuth2Redirect, redirect)

	client.Authorize(state, func(loginURL string) {
		ctx.RedirectTemporary(loginURL)
	})
	return true, nil
}

func (s *Service) HandleOAuthCallback(ctx *zoox.Context) (bool, error) {
	if s.Type() != "oauth" {
		return false, fmt.Errorf("oauth login is disabled")
	}
	code := ctx.Query().Get("code").String()
	state := ctx.Query().Get("state").String()
	if code == "" || state == "" {
		return false, fmt.Errorf("oauth callback: missing code or state")
	}
	expectedState := ctx.Session().Get(sessOAuth2State)
	if expectedState == "" || state != expectedState {
		return false, fmt.Errorf("oauth callback: state mismatch")
	}

	client, err := s.newOAuthClient(ctx)
	if err != nil {
		return false, err
	}

	var redirected bool
	var cbErr error
	client.Callback(code, state, func(user *gozooxoauth2.User, _ *gozooxoauth2.Token, err error) {
		if err != nil {
			cbErr = fmt.Errorf("oauth callback exchange failed: %w", err)
			return
		}
		username := oauthUsername(user)
		displayName := oauthDisplayName(user, username)
		userBytes, _ := json.Marshal(user)
		ctx.Session().Set(sessOAuth2User, string(userBytes))
		ctx.Session().Del(sessOAuth2State)
		s.setSessionUser(ctx, &UserInfo{Username: username, DisplayName: displayName})

		redirectURL := strings.TrimSpace(ctx.Session().Get(sessOAuth2Redirect))
		if redirectURL == "" {
			redirectURL = "/"
		}
		ctx.Session().Del(sessOAuth2Redirect)
		ctx.RedirectTemporary(redirectURL)
		redirected = true
	})
	if cbErr != nil {
		return false, cbErr
	}
	return redirected, nil
}

func (s *Service) newOAuthClient(ctx *zoox.Context) (gozooxoauth2.Client, error) {
	cfg := s.cfg.Auth.OAuth
	if !supportedOAuthProviders[cfg.Provider] {
		return nil, fmt.Errorf("unsupported oauth provider: %s", cfg.Provider)
	}
	redirectURL := strings.TrimSpace(cfg.RedirectURL)
	if redirectURL == "" {
		scheme := "http"
		if ctx.Request.TLS != nil {
			scheme = "https"
		}
		redirectURL = fmt.Sprintf("%s://%s%s", scheme, ctx.Host(), oauthCallbackPath)
	}
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = defaultOAuthScopes[cfg.Provider]
	}
	oauth2Cfg := &gozooxoauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURI:  redirectURL,
		Scope:        strings.Join(scopes, " "),
	}
	return create.Create(cfg.Provider, oauth2Cfg)
}

func oauthUsername(user *gozooxoauth2.User) string {
	if user == nil {
		return "oauth-user"
	}
	for _, v := range []string{user.Username, user.Email, user.Nickname, user.ID} {
		v = strings.TrimSpace(v)
		if v != "" {
			return strings.ToLower(v)
		}
	}
	return "oauth-user"
}

func oauthDisplayName(user *gozooxoauth2.User, fallback string) string {
	if user == nil {
		return fallback
	}
	for _, v := range []string{user.Nickname, user.Username, user.Email} {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return fallback
}

func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func fail(ctx *zoox.Context, status int, message string) {
	ctx.JSON(status, zoox.H{
		"code":    status,
		"message": message,
		"result":  nil,
	})
}

// HashPassword is used by RBAC seeding.
func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
