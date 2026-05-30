package core

import (
	"encoding/json"
	"mime"
	"net/http"
	"strconv"
	"strings"
)

const (
	errorPageContentTypeHTML = "text/html; charset=utf-8"
	errorPageContentTypeJSON = "application/json; charset=utf-8"
)

type errorPageJSONBody struct {
	Status  int    `json:"status"`
	Error   string `json:"error"`
	Message string `json:"message"`
	Reason  string `json:"reason,omitempty"`
	Host    string `json:"host,omitempty"`
	Path    string `json:"path,omitempty"`
	Method  string `json:"method,omitempty"`
}

// requestPrefersJSON reports whether the client prefers JSON over HTML for error responses.
// Browsers that send text/html in Accept continue to receive HTML pages; API clients that
// send Accept: application/json (without a higher-priority text/html) receive JSON.
func requestPrefersJSON(r *http.Request) bool {
	if r == nil {
		return false
	}
	accept := strings.TrimSpace(r.Header.Get("Accept"))
	if accept == "" {
		return false
	}

	jsonQ := -1.0
	htmlQ := -1.0
	for _, part := range strings.Split(accept, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		mediaType, params, err := mime.ParseMediaType(part)
		if err != nil {
			continue
		}
		q := 1.0
		if qs, ok := params["q"]; ok {
			if v, err := strconv.ParseFloat(qs, 64); err == nil {
				q = v
			}
		}
		switch strings.ToLower(mediaType) {
		case "application/json":
			if q > jsonQ {
				jsonQ = q
			}
		case "text/html":
			if q > htmlQ {
				htmlQ = q
			}
		}
	}

	if jsonQ < 0 {
		return false
	}
	if htmlQ < 0 {
		return true
	}
	return jsonQ > htmlQ
}

func ingressErrorPageJSON(status int, title, subtitle string, exposeDetails bool, hostname, path, method, reason string) string {
	payload := errorPageJSONBody{
		Status:  status,
		Error:   title,
		Message: subtitle,
	}
	if exposeDetails {
		payload.Reason = strings.TrimSpace(reason)
		payload.Host = hostname
		payload.Path = path
		payload.Method = strings.ToUpper(method)
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return `{"status":500,"error":"Internal Server Error","message":"An unexpected error occurred."}`
	}
	return string(b)
}

func inlineBodyLooksLikeJSON(body string) bool {
	trimmed := strings.TrimSpace(body)
	return strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")
}
