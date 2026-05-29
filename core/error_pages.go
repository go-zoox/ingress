package core

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-zoox/ingress/core/security"
	"github.com/go-zoox/zoox"
	"github.com/go-zoox/zoox/middleware"
)

// Supported configurable HTML error page status codes.
var supportedErrorPageStatuses = []int{
	http.StatusUnauthorized,
	http.StatusForbidden,
	http.StatusNotFound,
	http.StatusInternalServerError,
	http.StatusBadGateway,
	http.StatusServiceUnavailable,
	http.StatusGatewayTimeout,
}

const (
	errorPageTypeBuiltin = "builtin"
	errorPageTypeFile    = "file"
	errorPageTypeInline  = "inline"
)

// ErrorPages configures built-in and custom HTML responses for common HTTP errors.
type ErrorPages struct {
	Pages map[string]ErrorPageSpec `config:"pages"`
}

// ErrorPageSpec selects a built-in template or custom HTML for one status code.
type ErrorPageSpec struct {
	// Type: builtin (default), file, or inline.
	Type string `config:"type,default=builtin"`
	// File is a path to an HTML document (relative to ingress.yaml when not absolute).
	File string `config:"file"`
	// Body is inline HTML when type is inline.
	Body string `config:"body"`
	// Title overrides the built-in heading when type is builtin.
	Title string `config:"title"`
	// Subtitle overrides the built-in message when type is builtin.
	Subtitle string `config:"subtitle"`
}

// ErrorPageDetail optional request context for verbose built-in pages.
type ErrorPageDetail struct {
	Hostname string
	Path     string
	Method   string
	Reason   string
}

type compiledErrorPage struct {
	status   int
	pageType string
	body     string
	title    string
	subtitle string
}

type compiledErrorPages struct {
	byStatus map[int]*compiledErrorPage
}

func compileErrorPages(cfg *Config) (*compiledErrorPages, error) {
	out := &compiledErrorPages{byStatus: make(map[int]*compiledErrorPage)}
	overrides := map[int]ErrorPageSpec{}
	if cfg != nil && cfg.ErrorPages.Pages != nil {
		for codeStr, spec := range cfg.ErrorPages.Pages {
			code, err := parseErrorPageStatus(codeStr)
			if err != nil {
				return nil, fmt.Errorf("error_pages.pages[%q]: %w", codeStr, err)
			}
			overrides[code] = spec
		}
	}

	for _, status := range supportedErrorPageStatuses {
		spec, ok := overrides[status]
		if !ok {
			title, subtitle := builtinErrorPageCopy(status)
			out.byStatus[status] = &compiledErrorPage{
				status:   status,
				pageType: errorPageTypeBuiltin,
				title:    title,
				subtitle: subtitle,
			}
			continue
		}

		pageType := strings.ToLower(strings.TrimSpace(spec.Type))
		if pageType == "" {
			pageType = errorPageTypeBuiltin
		}
		switch pageType {
		case errorPageTypeBuiltin:
			title, subtitle := builtinErrorPageCopy(status)
			if t := strings.TrimSpace(spec.Title); t != "" {
				title = t
			}
			if s := strings.TrimSpace(spec.Subtitle); s != "" {
				subtitle = s
			}
			out.byStatus[status] = &compiledErrorPage{
				status:   status,
				pageType: errorPageTypeBuiltin,
				title:    title,
				subtitle: subtitle,
			}
		case errorPageTypeFile:
			path := strings.TrimSpace(spec.File)
			if path == "" {
				return nil, fmt.Errorf("error_pages.pages[%d].file is required when type is file", status)
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("error_pages.pages[%d].file: %w", status, err)
			}
			out.byStatus[status] = &compiledErrorPage{
				status:   status,
				pageType: errorPageTypeFile,
				body:     string(content),
			}
		case errorPageTypeInline:
			body := spec.Body
			if strings.TrimSpace(body) == "" {
				return nil, fmt.Errorf("error_pages.pages[%d].body is required when type is inline", status)
			}
			out.byStatus[status] = &compiledErrorPage{
				status:   status,
				pageType: errorPageTypeInline,
				body:     body,
			}
		default:
			return nil, fmt.Errorf("error_pages.pages[%d].type %q is invalid (builtin, file, inline)", status, spec.Type)
		}
	}

	return out, nil
}

func parseErrorPageStatus(codeStr string) (int, error) {
	codeStr = strings.TrimSpace(codeStr)
	if codeStr == "" {
		return 0, fmt.Errorf("status code is required")
	}
	code, err := strconv.Atoi(codeStr)
	if err != nil {
		return 0, fmt.Errorf("invalid status code %q", codeStr)
	}
	for _, allowed := range supportedErrorPageStatuses {
		if code == allowed {
			return code, nil
		}
	}
	return 0, fmt.Errorf("unsupported status code %d (allowed: 401, 403, 404, 500, 502, 503, 504)", code)
}

func validateErrorPages(cfg *Config) error {
	if cfg == nil {
		return nil
	}
	for codeStr, spec := range cfg.ErrorPages.Pages {
		code, err := parseErrorPageStatus(codeStr)
		if err != nil {
			return fmt.Errorf("error_pages.pages[%q]: %w", codeStr, err)
		}
		pageType := strings.ToLower(strings.TrimSpace(spec.Type))
		if pageType == "" {
			pageType = errorPageTypeBuiltin
		}
		switch pageType {
		case errorPageTypeBuiltin:
		case errorPageTypeFile:
			if strings.TrimSpace(spec.File) == "" {
				return fmt.Errorf("error_pages.pages[%d].file is required when type is file", code)
			}
		case errorPageTypeInline:
			if strings.TrimSpace(spec.Body) == "" {
				return fmt.Errorf("error_pages.pages[%d].body is required when type is inline", code)
			}
		default:
			return fmt.Errorf("error_pages.pages[%d].type %q is invalid", code, spec.Type)
		}
		if strings.TrimSpace(spec.File) != "" && strings.TrimSpace(spec.Body) != "" && pageType != errorPageTypeFile {
			return fmt.Errorf("error_pages.pages[%d]: set either file or body, not both", code)
		}
	}
	return nil
}

func resolveErrorPagePaths(cfg *Config, base string) {
	if cfg == nil {
		return
	}
	for codeStr, spec := range cfg.ErrorPages.Pages {
		if strings.TrimSpace(spec.File) == "" {
			continue
		}
		spec.File = resolveConfigFilePath(base, spec.File)
		cfg.ErrorPages.Pages[codeStr] = spec
	}
}

func (p *compiledErrorPages) Render(status int, exposeDetails bool, detail ErrorPageDetail) string {
	page := p.byStatus[status]
	if page == nil {
		title, subtitle := builtinErrorPageCopy(status)
		return ingressErrorPageHTML(status, title, subtitle, exposeDetails, detail.Hostname, detail.Path, detail.Method, detail.Reason)
	}
	switch page.pageType {
	case errorPageTypeFile, errorPageTypeInline:
		return page.body
	default:
		host, path, method, reason := detail.Hostname, detail.Path, detail.Method, detail.Reason
		if !exposeDetails {
			host, path, method, reason = "", "", "", ""
		}
		return ingressErrorPageHTML(status, page.title, page.subtitle, exposeDetails, host, path, method, reason)
	}
}

func (c *core) writeErrorPage(ctx *zoox.Context, status int, secProf *security.Profile, detail ErrorPageDetail) {
	applySecurityHeaders(ctx, secProf)
	html := c.errorPages.Render(status, c.cfg.ErrorPageExposeDetails, detail)
	ctx.SetHeader("Content-Type", "text/html; charset=utf-8")
	ctx.HTML(status, html)
}

func (c *core) fillProxyErrorPages(cfg *middleware.ProxyConfig, detail ErrorPageDetail) {
	if c.errorPages == nil {
		return
	}
	safe := ErrorPageDetail{}
	if c.cfg.ErrorPageExposeDetails {
		safe = detail
	}
	cfg.ErrorPages.NotFound = c.errorPages.Render(http.StatusNotFound, c.cfg.ErrorPageExposeDetails, safe)
	cfg.ErrorPages.InternalServiceError = c.errorPages.Render(http.StatusInternalServerError, c.cfg.ErrorPageExposeDetails, safe)
	cfg.ErrorPages.BadGateway = c.errorPages.Render(http.StatusBadGateway, c.cfg.ErrorPageExposeDetails, safe)
	cfg.ErrorPages.ServiceUnavailable = c.errorPages.Render(http.StatusServiceUnavailable, c.cfg.ErrorPageExposeDetails, safe)
	cfg.ErrorPages.GatewayTimeout = c.errorPages.Render(http.StatusGatewayTimeout, c.cfg.ErrorPageExposeDetails, safe)
}

func shouldUseWAFErrorPage(status int, contentType, body string) bool {
	if status != http.StatusForbidden {
		return false
	}
	if strings.TrimSpace(contentType) != "text/plain; charset=utf-8" {
		return false
	}
	return strings.TrimSpace(body) == "Forbidden"
}

func builtinErrorPageCopy(status int) (title, subtitle string) {
	switch status {
	case http.StatusUnauthorized:
		return "Unauthorized", "Authentication is required to access this resource."
	case http.StatusForbidden:
		return "Forbidden", "You do not have permission to access this resource."
	case http.StatusNotFound:
		return "Not Found", "The requested resource could not be found."
	case http.StatusInternalServerError:
		return "Internal Server Error", "An unexpected error occurred. Please try again later."
	case http.StatusBadGateway:
		return "Bad Gateway", "The upstream server returned an invalid response."
	case http.StatusServiceUnavailable:
		return "Service Unavailable", "The service is temporarily unavailable. Please try again later."
	case http.StatusGatewayTimeout:
		return "Gateway Timeout", "The upstream server did not respond in time."
	default:
		return "Error", "An error occurred while processing your request."
	}
}
