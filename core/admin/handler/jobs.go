package handler

import (
	"net/http"
	"strconv"
	"strings"

	jobsvc "github.com/go-zoox/ingress/core/admin/service/jobs"
	ingjobs "github.com/go-zoox/ingress/core/jobs"
	"github.com/go-zoox/zoox"
)

// JobsHandler serves scheduled job APIs.
type JobsHandler struct {
	jobs *jobsvc.Service
}

func NewJobsHandler(jobs *jobsvc.Service) *JobsHandler {
	return &JobsHandler{jobs: jobs}
}

func (h *JobsHandler) Mount(g *zoox.RouterGroup) {
	g.Get("/jobs", h.List)
	g.Get("/jobs/capabilities", h.Capabilities)
	g.Get("/jobs/runs", h.ListRuns)
	g.Get("/jobs/runs/:id", h.GetRun)
	g.Put("/jobs/builtins/:id", h.UpdateBuiltin)
	g.Post("/jobs/items", h.CreateItem)
	g.Put("/jobs/items/:id", h.UpdateItem)
	g.Delete("/jobs/items/:id", h.DeleteItem)
	g.Post("/jobs/:source/:id/run", h.RunNow)
	g.Get("/jobs/:source/:id/runs", h.ListJobRuns)
}

func (h *JobsHandler) List(ctx *zoox.Context) {
	if h.jobs == nil {
		ok(ctx, jobsvc.ListResult{Capabilities: jobsvc.Capabilities{HTTPCall: true}})
		return
	}
	result, err := h.jobs.List()
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ok(ctx, result)
}

func (h *JobsHandler) Capabilities(ctx *zoox.Context) {
	if h.jobs == nil {
		ok(ctx, jobsvc.Capabilities{HTTPCall: true})
		return
	}
	ok(ctx, h.jobs.Capabilities())
}

func (h *JobsHandler) ListRuns(ctx *zoox.Context) {
	if h.jobs == nil {
		ok(ctx, []jobsvc.RunRow{})
		return
	}
	limit, _ := strconv.Atoi(strings.TrimSpace(ctx.Query().Get("limit").String()))
	rows, err := h.jobs.ListRuns(strings.TrimSpace(ctx.Query().Get("job_id").String()), limit)
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ok(ctx, rows)
}

func (h *JobsHandler) GetRun(ctx *zoox.Context) {
	if h.jobs == nil {
		fail(ctx, http.StatusServiceUnavailable, "jobs service unavailable")
		return
	}
	id, err := strconv.ParseUint(strings.TrimSpace(ctx.Param().Get("id").String()), 10, 64)
	if err != nil || id == 0 {
		fail(ctx, http.StatusBadRequest, "invalid run id")
		return
	}
	row, err := h.jobs.GetRun(uint(id))
	if err != nil {
		fail(ctx, http.StatusNotFound, err.Error())
		return
	}
	ok(ctx, row)
}

func (h *JobsHandler) UpdateBuiltin(ctx *zoox.Context) {
	if h.jobs == nil {
		fail(ctx, http.StatusServiceUnavailable, "jobs service unavailable")
		return
	}
	var body jobsvc.BuiltinPatch
	if err := ctx.BindJSON(&body); err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.jobs.UpdateBuiltin(ctx.Param().Get("id").String(), body); err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ok(ctx, zoox.H{"ok": true})
}

func (h *JobsHandler) CreateItem(ctx *zoox.Context) {
	if h.jobs == nil {
		fail(ctx, http.StatusServiceUnavailable, "jobs service unavailable")
		return
	}
	var body ingjobs.Item
	if err := ctx.BindJSON(&body); err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.jobs.CreateItem(body); err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ok(ctx, zoox.H{"ok": true})
}

func (h *JobsHandler) UpdateItem(ctx *zoox.Context) {
	if h.jobs == nil {
		fail(ctx, http.StatusServiceUnavailable, "jobs service unavailable")
		return
	}
	var body ingjobs.Item
	if err := ctx.BindJSON(&body); err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.jobs.UpdateItem(ctx.Param().Get("id").String(), body); err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ok(ctx, zoox.H{"ok": true})
}

func (h *JobsHandler) DeleteItem(ctx *zoox.Context) {
	if h.jobs == nil {
		fail(ctx, http.StatusServiceUnavailable, "jobs service unavailable")
		return
	}
	if err := h.jobs.DeleteItem(ctx.Param().Get("id").String()); err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ok(ctx, zoox.H{"ok": true})
}

func (h *JobsHandler) RunNow(ctx *zoox.Context) {
	if h.jobs == nil {
		fail(ctx, http.StatusServiceUnavailable, "jobs service unavailable")
		return
	}
	source := ctx.Param().Get("source").String()
	id := ctx.Param().Get("id").String()
	row, err := h.jobs.RunNow(source, id)
	if err != nil {
		fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ok(ctx, row)
}

func (h *JobsHandler) ListJobRuns(ctx *zoox.Context) {
	if h.jobs == nil {
		ok(ctx, []jobsvc.RunRow{})
		return
	}
	limit, _ := strconv.Atoi(strings.TrimSpace(ctx.Query().Get("limit").String()))
	rows, err := h.jobs.ListRuns(ctx.Param().Get("id").String(), limit)
	if err != nil {
		fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ok(ctx, rows)
}
