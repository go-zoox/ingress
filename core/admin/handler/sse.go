package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-zoox/ingress/core/admin/service"
	"github.com/go-zoox/zoox"
)

// SSEHandler handles Server-Sent Events streaming.
type SSEHandler struct {
	broker   *service.SSEBroker
	overview *service.OverviewStreamer
}

// NewSSEHandler creates a new SSE handler.
func NewSSEHandler(broker *service.SSEBroker, overview *service.OverviewStreamer) *SSEHandler {
	return &SSEHandler{broker: broker, overview: overview}
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

	sub, err := h.broker.Subscribe(cleanChannels, clientIP, params)
	if err != nil {
		fail(ctx, http.StatusTooManyRequests, err.Error())
		return
	}
	wantsOverview := false
	for _, c := range cleanChannels {
		if c == "overview" {
			wantsOverview = true
			break
		}
	}
	defer func() {
		if wantsOverview && h.overview != nil {
			h.overview.ForgetSubscriber(sub)
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
