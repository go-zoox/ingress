package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-zoox/ingress/core/admin/service"
	"github.com/go-zoox/zoox"
)

// SSEHandler handles Server-Sent Events streaming.
type SSEHandler struct {
	broker        *service.SSEBroker
	overview      *service.OverviewStreamer
	detailMetrics *service.DetailMetricsStreamer
}

// NewSSEHandler creates a new SSE handler.
func NewSSEHandler(broker *service.SSEBroker, overview *service.OverviewStreamer, detail *service.DetailMetricsStreamer) *SSEHandler {
	return &SSEHandler{broker: broker, overview: overview, detailMetrics: detail}
}

func writeSSEEvent(w http.ResponseWriter, flusher http.Flusher, event service.SSEEvent) {
	if event.Event != "" {
		fmt.Fprintf(w, "event: %s\n", event.Event)
	}
	if event.ID != "" {
		fmt.Fprintf(w, "id: %s\n", event.ID)
	}
	if event.Retry > 0 {
		fmt.Fprintf(w, "retry: %d\n", event.Retry)
	}
	fmt.Fprintf(w, "data: %s\n\n", event.Data)
	if flusher != nil {
		flusher.Flush()
	}
}

// Stream handles GET /api/v1/events/stream?channels=metrics,waf,logs,health,overview
func (h *SSEHandler) Stream(ctx *zoox.Context) {
	channelsParam := strings.TrimSpace(ctx.Query().Get("channels").String())
	if channelsParam == "" {
		fail(ctx, http.StatusBadRequest, "channels parameter is required")
		return
	}
	channels := strings.Split(channelsParam, ",")
	var cleanChannels []string
	for _, c := range channels {
		c = strings.TrimSpace(c)
		if c != "" {
			cleanChannels = append(cleanChannels, c)
		}
	}
	if len(cleanChannels) == 0 {
		fail(ctx, http.StatusBadRequest, "at least one channel is required")
		return
	}

	clientIP := ctx.ClientIP()
	if clientIP == "" {
		clientIP = ctx.Request.RemoteAddr
	}

	params := map[string]string{}
	if window := strings.TrimSpace(ctx.Query().Get("window").String()); window != "" {
		params["window"] = window
	}
	if from := strings.TrimSpace(ctx.Query().Get("from").String()); from != "" {
		params["from"] = from
	}
	if to := strings.TrimSpace(ctx.Query().Get("to").String()); to != "" {
		params["to"] = to
	}
	if ri := strings.TrimSpace(ctx.Query().Get("ri").String()); ri != "" {
		params["ri"] = ri
	}
	if pi := strings.TrimSpace(ctx.Query().Get("pi").String()); pi != "" {
		params["pi"] = pi
	}
	if name := strings.TrimSpace(ctx.Query().Get("name").String()); name != "" {
		if decoded, err := url.PathUnescape(name); err == nil {
			name = decoded
		}
		params["name"] = name
	}
	if host := strings.TrimSpace(ctx.Query().Get("host").String()); host != "" {
		params["host"] = host
	}
	if path := strings.TrimSpace(ctx.Query().Get("path").String()); path != "" {
		params["path"] = path
	}
	if pathMatch := strings.TrimSpace(ctx.Query().Get("path_match").String()); pathMatch != "" {
		params["path_match"] = pathMatch
	}

	sub, err := h.broker.Subscribe(cleanChannels, clientIP, params)
	if err != nil {
		fail(ctx, http.StatusTooManyRequests, err.Error())
		return
	}
	wantsOverview := false
	wantsRouteMetrics := false
	wantsServiceMetrics := false
	for _, c := range cleanChannels {
		switch c {
		case "overview":
			wantsOverview = true
		case "route_metrics":
			wantsRouteMetrics = true
		case "service_metrics":
			wantsServiceMetrics = true
		}
	}
	defer func() {
		if h.overview != nil && wantsOverview {
			h.overview.ForgetSubscriber(sub)
		}
		if h.detailMetrics != nil && (wantsRouteMetrics || wantsServiceMetrics) {
			h.detailMetrics.ForgetSubscriber(sub)
		}
		h.broker.Unsubscribe(sub, clientIP)
	}()

	w := ctx.Writer
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	flusher, canFlush := w.(http.Flusher)

	fmt.Fprintf(w, "event: connected\ndata: {\"channels\":[%q]}\n\n", channelsParam)
	if canFlush {
		flusher.Flush()
	}

	if wantsOverview && h.overview != nil {
		writeSSEEvent(w, flusher, h.overview.SnapshotEvent(sub))
	}
	if wantsRouteMetrics && h.detailMetrics != nil {
		writeSSEEvent(w, flusher, h.detailMetrics.RouteSnapshotEvent(sub))
	}
	if wantsServiceMetrics && h.detailMetrics != nil {
		writeSSEEvent(w, flusher, h.detailMetrics.ServiceSnapshotEvent(sub))
	}

	req := ctx.Request
	done := req.Context().Done()

	for {
		select {
		case <-done:
			return
		case event, ok := <-sub.Ch:
			if !ok {
				return
			}
			writeSSEEvent(w, flusher, event)
		}
	}
}
