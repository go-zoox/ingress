package handler

import (
	"net/http"
	"strconv"
	"strings"

	adminauth "github.com/go-zoox/ingress/core/admin/auth"
	"github.com/go-zoox/ingress/core/admin/service"
	"github.com/go-zoox/zoox"
)

// WAFBanHandler serves the runtime ban list API.
type WAFBanHandler struct {
	bans *service.WAFBanService
	auth *adminauth.Service
}

// NewWAFBanHandler builds the handler.
func NewWAFBanHandler(bans *service.WAFBanService, auth *adminauth.Service) *WAFBanHandler {
	return &WAFBanHandler{bans: bans, auth: auth}
}

// Mount registers the ban endpoints on the /api/v1 group.
func (h *WAFBanHandler) Mount(g *zoox.RouterGroup) {
	g.Get("/waf/bans", h.List)
	g.Post("/waf/bans", h.Create)
	g.Get("/waf/bans/:id", h.Get)
	g.Patch("/waf/bans/:id", h.Update)
	g.Delete("/waf/bans/:id", h.Delete)
	g.Post("/waf/bans/preview", h.Preview)
}

// actor resolves the logged-in username, falling back to "admin" for
// unauthenticated deployments (auth disabled).
func (h *WAFBanHandler) actor(ctx *zoox.Context) string {
	if h.auth != nil {
		if name := h.auth.UsernameFromContext(ctx); name != "" {
			return name
		}
	}
	return "admin"
}

// List handles GET /api/v1/waf/bans.
func (h *WAFBanHandler) List(ctx *zoox.Context) {
	limit, _ := strconv.Atoi(ctx.Query().Get("limit").String())
	rows, err := h.bans.List(service.BanFilter{
		Status:  ctx.Query().Get("status").String(),
		Kind:    ctx.Query().Get("kind").String(),
		Keyword: ctx.Query().Get("keyword").String(),
		Limit:   limit,
	})
	if err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ok(ctx, rows)
}

// Get handles GET /api/v1/waf/bans/:id.
func (h *WAFBanHandler) Get(ctx *zoox.Context) {
	id, err := parseUintParam(ctx, "id")
	if err != nil {
		fail(ctx, http.StatusBadRequest, "invalid ban id")
		return
	}
	row, err := h.bans.Get(id)
	if err != nil {
		fail(ctx, statusForBanError(err), err.Error())
		return
	}
	ok(ctx, row)
}

// Create handles POST /api/v1/waf/bans.
func (h *WAFBanHandler) Create(ctx *zoox.Context) {
	var body service.BanInput
	if err := ctx.BindJSON(&body); err != nil {
		fail(ctx, http.StatusBadRequest, "invalid json body")
		return
	}
	body.Actor = h.actor(ctx)

	row, err := h.bans.Create(body)
	if err != nil {
		fail(ctx, statusForBanError(err), err.Error())
		return
	}
	ok(ctx, row)
}

// Update handles PATCH /api/v1/waf/bans/:id.
func (h *WAFBanHandler) Update(ctx *zoox.Context) {
	id, err := parseUintParam(ctx, "id")
	if err != nil {
		fail(ctx, http.StatusBadRequest, "invalid ban id")
		return
	}
	var body service.BanPatch
	if err := ctx.BindJSON(&body); err != nil {
		fail(ctx, http.StatusBadRequest, "invalid json body")
		return
	}
	body.Actor = h.actor(ctx)

	row, err := h.bans.Update(id, body)
	if err != nil {
		fail(ctx, statusForBanError(err), err.Error())
		return
	}
	ok(ctx, row)
}

// Delete handles DELETE /api/v1/waf/bans/:id.
func (h *WAFBanHandler) Delete(ctx *zoox.Context) {
	id, err := parseUintParam(ctx, "id")
	if err != nil {
		fail(ctx, http.StatusBadRequest, "invalid ban id")
		return
	}
	if err := h.bans.Delete(id, h.actor(ctx)); err != nil {
		fail(ctx, statusForBanError(err), err.Error())
		return
	}
	ok(ctx, zoox.H{"ok": true})
}

// Preview handles POST /api/v1/waf/bans/preview.
func (h *WAFBanHandler) Preview(ctx *zoox.Context) {
	var body service.BanInput
	if err := ctx.BindJSON(&body); err != nil {
		fail(ctx, http.StatusBadRequest, "invalid json body")
		return
	}
	out, err := h.bans.Preview(body)
	if err != nil {
		fail(ctx, statusForBanError(err), err.Error())
		return
	}
	ok(ctx, out)
}

// statusForBanError maps service error prefixes onto HTTP statuses. The message
// prefixes are the contract documented in the change design; the envelope stays
// the shared {code, message, result}.
func statusForBanError(err error) int {
	if err == nil {
		return http.StatusOK
	}
	msg := err.Error()
	switch {
	case strings.HasPrefix(msg, "not_found:"):
		return http.StatusNotFound
	case strings.HasPrefix(msg, "trusted_proxy:"),
		strings.HasPrefix(msg, "confirm_required:"),
		strings.HasPrefix(msg, "invalid_value:"),
		strings.HasPrefix(msg, "invalid_kind:"),
		strings.HasPrefix(msg, "invalid_hosts:"),
		strings.HasPrefix(msg, "invalid_ttl:"),
		strings.HasPrefix(msg, "duplicate:"):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
