package handler

import (
	"errors"
	"net/http"
	"strings"

	adminauth "github.com/go-zoox/ingress/core/admin/auth"
	"github.com/go-zoox/zoox"
)

// AuthHandler serves Admin Console authentication APIs.
type AuthHandler struct {
	auth *adminauth.Service
}

func NewAuthHandler(auth *adminauth.Service) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) Mount(g *zoox.RouterGroup) {
	g.Get("/auth/config", h.Config)
	g.Post("/auth/login", h.Login)
	g.Post("/auth/logout", h.Logout)
	g.Get("/auth/oauth/login", h.OAuthLogin)
	g.Get("/auth/oauth/callback", h.OAuthCallback)
}

func (h *AuthHandler) Config(ctx *zoox.Context) {
	ok(ctx, h.auth.ConfigView(ctx))
}

func (h *AuthHandler) Login(ctx *zoox.Context) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := ctx.BindJSON(&body); err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	user, err := h.auth.LoginBasic(ctx, body.Username, body.Password)
	if err != nil {
		if errors.Is(err, adminauth.ErrNoConsoleAccess) {
			fail(ctx, http.StatusForbidden, err.Error())
			return
		}
		fail(ctx, http.StatusUnauthorized, err.Error())
		return
	}
	ok(ctx, zoox.H{"ok": true, "user": user})
}

func (h *AuthHandler) Logout(ctx *zoox.Context) {
	h.auth.Logout(ctx)
	ok(ctx, zoox.H{"ok": true})
}

func (h *AuthHandler) OAuthLogin(ctx *zoox.Context) {
	redirect := strings.TrimSpace(ctx.Query().Get("redirect").String())
	redirected, err := h.auth.StartOAuth(ctx, redirect)
	if err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if !redirected {
		fail(ctx, http.StatusInternalServerError, "oauth redirect failed")
	}
}

func (h *AuthHandler) OAuthCallback(ctx *zoox.Context) {
	redirected, err := h.auth.HandleOAuthCallback(ctx)
	if err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if !redirected {
		fail(ctx, http.StatusInternalServerError, "oauth callback failed")
	}
}
