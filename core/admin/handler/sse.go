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
	broker *service.SSEBroker
}

// NewSSEHandler creates a new SSE handler.
func NewSSEHandler(broker *service.SSEBroker) *SSEHandler {
	return &SSEHandler{broker: broker}
}

// Stream handles GET /api/v1/events/stream?channels=metrics,waf,logs,health
func (h *SSEHandler) Stream(ctx *zoox.Context) {
	// Parse requested channels from query param
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

	// Get client IP for connection limiting
	clientIP := ctx.ClientIP()
	if clientIP == "" {
		clientIP = ctx.Request.RemoteAddr
	}

	// Subscribe to the requested channels
	sub, err := h.broker.Subscribe(cleanChannels, clientIP)
	if err != nil {
		fail(ctx, http.StatusTooManyRequests, err.Error())
		return
	}
	defer h.broker.Unsubscribe(sub, clientIP)

	// Set SSE response headers
	w := ctx.Writer
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	flusher, canFlush := w.(http.Flusher)
	_ = flusher
	_ = canFlush

	// Send initial connection established event
	fmt.Fprintf(w, "event: connected\ndata: {\"channels\":[%q]}\n\n", channelsParam)
	if canFlush {
		flusher.Flush()
	}

	// Stream events to the client
	req := ctx.Request
	done := req.Context().Done()

	for {
		select {
		case <-done:
			// Client disconnected
			return
		case event, ok := <-sub.Ch:
			if !ok {
				return
			}
			// Write SSE event
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
			if canFlush {
				flusher.Flush()
			}
		}
	}
}
