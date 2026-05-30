package handler

import (
	"net/http"
	"strconv"
	"strings"

	rbacsvc "github.com/go-zoox/ingress/core/admin/service/rbac"
	adminauth "github.com/go-zoox/ingress/core/admin/auth"
	"github.com/go-zoox/zoox"
)

// RBACHandler serves RBAC management APIs.
type RBACHandler struct {
	rbac *rbacsvc.Service
	auth *adminauth.Service
}

func NewRBACHandler(rbac *rbacsvc.Service, auth *adminauth.Service) *RBACHandler {
	return &RBACHandler{rbac: rbac, auth: auth}
}

func (h *RBACHandler) Mount(g *zoox.RouterGroup) {
	g.Get("/rbac/menus", h.ListMenus)
	g.Get("/rbac/permissions", h.ListPermissions)
	g.Post("/rbac/permissions", h.CreatePermission)
	g.Put("/rbac/permissions/:id", h.UpdatePermission)
	g.Delete("/rbac/permissions/:id", h.DeletePermission)

	g.Get("/rbac/roles", h.ListRoles)
	g.Post("/rbac/roles", h.CreateRole)
	g.Put("/rbac/roles/:id", h.UpdateRole)
	g.Delete("/rbac/roles/:id", h.DeleteRole)

	g.Get("/rbac/users", h.ListUsers)
	g.Post("/rbac/users", h.CreateUser)
	g.Put("/rbac/users/:id", h.UpdateUser)
	g.Put("/rbac/users/:id/password", h.UpdateUserPassword)
	g.Delete("/rbac/users/:id", h.DeleteUser)
}

func (h *RBACHandler) ListMenus(ctx *zoox.Context) {
	username := ""
	if h.auth != nil {
		username = h.auth.UsernameFromContext(ctx)
	}
	if username == "" {
		username = strings.TrimSpace(ctx.Query().Get("username").String())
	}
	out, err := h.rbac.ListNavigation(username)
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	if out.Groups == nil {
		out.Groups = []rbacsvc.NavGroupRow{}
	}
	ok(ctx, out)
}

func (h *RBACHandler) ListPermissions(ctx *zoox.Context) {
	rows, err := h.rbac.ListPermissions()
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	if rows == nil {
		rows = []rbacsvc.PermissionRow{}
	}
	ok(ctx, rows)
}

func (h *RBACHandler) CreatePermission(ctx *zoox.Context) {
	var body rbacsvc.PermissionInput
	if err := ctx.BindJSON(&body); err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	row, err := h.rbac.CreatePermission(body)
	if err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ok(ctx, row)
}

func (h *RBACHandler) UpdatePermission(ctx *zoox.Context) {
	id, err := parseUintParam(ctx, "id")
	if err != nil {
		fail(ctx, http.StatusBadRequest, "invalid permission id")
		return
	}
	var body rbacsvc.PermissionInput
	if err := ctx.BindJSON(&body); err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	row, err := h.rbac.UpdatePermission(id, body)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			fail(ctx, http.StatusNotFound, err.Error())
			return
		}
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ok(ctx, row)
}

func (h *RBACHandler) DeletePermission(ctx *zoox.Context) {
	id, err := parseUintParam(ctx, "id")
	if err != nil {
		fail(ctx, http.StatusBadRequest, "invalid permission id")
		return
	}
	if err := h.rbac.DeletePermission(id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			fail(ctx, http.StatusNotFound, err.Error())
			return
		}
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ok(ctx, zoox.H{"ok": true})
}

func (h *RBACHandler) ListRoles(ctx *zoox.Context) {
	rows, err := h.rbac.ListRoles()
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	if rows == nil {
		rows = []rbacsvc.RoleRow{}
	}
	ok(ctx, rows)
}

func (h *RBACHandler) CreateRole(ctx *zoox.Context) {
	var body rbacsvc.RoleInput
	if err := ctx.BindJSON(&body); err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	row, err := h.rbac.CreateRole(body)
	if err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ok(ctx, row)
}

func (h *RBACHandler) UpdateRole(ctx *zoox.Context) {
	id, err := parseUintParam(ctx, "id")
	if err != nil {
		fail(ctx, http.StatusBadRequest, "invalid role id")
		return
	}
	var body rbacsvc.RoleInput
	if err := ctx.BindJSON(&body); err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	row, err := h.rbac.UpdateRole(id, body)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			fail(ctx, http.StatusNotFound, err.Error())
			return
		}
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ok(ctx, row)
}

func (h *RBACHandler) DeleteRole(ctx *zoox.Context) {
	id, err := parseUintParam(ctx, "id")
	if err != nil {
		fail(ctx, http.StatusBadRequest, "invalid role id")
		return
	}
	if err := h.rbac.DeleteRole(id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			fail(ctx, http.StatusNotFound, err.Error())
			return
		}
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ok(ctx, zoox.H{"ok": true})
}

func (h *RBACHandler) ListUsers(ctx *zoox.Context) {
	rows, err := h.rbac.ListUsers()
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	if rows == nil {
		rows = []rbacsvc.UserRow{}
	}
	ok(ctx, rows)
}

func (h *RBACHandler) CreateUser(ctx *zoox.Context) {
	var body rbacsvc.UserInput
	if err := ctx.BindJSON(&body); err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	row, err := h.rbac.CreateUser(body)
	if err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ok(ctx, row)
}

func (h *RBACHandler) UpdateUser(ctx *zoox.Context) {
	id, err := parseUintParam(ctx, "id")
	if err != nil {
		fail(ctx, http.StatusBadRequest, "invalid user id")
		return
	}
	var body rbacsvc.UserInput
	if err := ctx.BindJSON(&body); err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	row, err := h.rbac.UpdateUser(id, body)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			fail(ctx, http.StatusNotFound, err.Error())
			return
		}
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ok(ctx, row)
}

func (h *RBACHandler) UpdateUserPassword(ctx *zoox.Context) {
	id, err := parseUintParam(ctx, "id")
	if err != nil {
		fail(ctx, http.StatusBadRequest, "invalid user id")
		return
	}
	var body rbacsvc.PasswordInput
	if err := ctx.BindJSON(&body); err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.rbac.UpdateUserPassword(id, body); err != nil {
		if strings.Contains(err.Error(), "not found") {
			fail(ctx, http.StatusNotFound, err.Error())
			return
		}
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ok(ctx, zoox.H{"ok": true})
}

func (h *RBACHandler) DeleteUser(ctx *zoox.Context) {
	id, err := parseUintParam(ctx, "id")
	if err != nil {
		fail(ctx, http.StatusBadRequest, "invalid user id")
		return
	}
	if err := h.rbac.DeleteUser(id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			fail(ctx, http.StatusNotFound, err.Error())
			return
		}
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ok(ctx, zoox.H{"ok": true})
}

func parseUintParam(ctx *zoox.Context, name string) (uint, error) {
	id, err := strconv.ParseUint(strings.TrimSpace(ctx.Param().Get(name).String()), 10, 64)
	if err != nil || id == 0 {
		return 0, err
	}
	return uint(id), nil
}
