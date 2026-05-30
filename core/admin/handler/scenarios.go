package handler

import (
	"net/http"
	"strings"

	"github.com/go-zoox/zoox"
)

func (a *API) Scenarios(ctx *zoox.Context) {
	out, err := a.scenarios.List()
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ok(ctx, out)
}

func (a *API) SetScenarioActive(ctx *zoox.Context) {
	var body struct {
		ID string `json:"id"`
	}
	if err := ctx.BindJSON(&body); err != nil {
		fail(ctx, http.StatusBadRequest, "invalid json body")
		return
	}
	id := strings.TrimSpace(body.ID)
	if id == "" {
		fail(ctx, http.StatusBadRequest, "id is required")
		return
	}
	out, err := a.scenarios.SetActive(id)
	if err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ok(ctx, out)
}
