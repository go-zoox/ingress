package core

import (
	"context"
	"html/template"
	"fmt"
	"mime"
	"net/http"
	"os"
	pathlib "path"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/logger"
	"github.com/go-zoox/proxy"
	"github.com/go-zoox/zoox"
	"github.com/go-zoox/zoox/middleware"
)

func (c *core) build() error {
	// config
	c.app.Config.Port = int(c.cfg.Port)
	c.app.Config.HTTPSPort = int(c.cfg.HTTPS.Port)

	// middlewares
	c.app.Use(func(ctx *zoox.Context) {
		if c.cfg.HealthCheck.Outer.Enable {
			if ctx.Path == c.cfg.HealthCheck.Outer.Path {
				if c.cfg.HealthCheck.Outer.Ok {
					ctx.String(http.StatusOK, "ok")
					return
				}
			}
		}

		ctx.Next()
	})

	// services (core plugin)
	c.app.Use(middleware.Proxy(func(ctx *zoox.Context, cfg *middleware.ProxyConfig) (next, stop bool, err error) {
		hostname := ctx.Hostname()
		method := ctx.Method
		path := ctx.Path

		serviceIns, matchedRule, pathBackend, err := c.match(ctx, hostname, path)
		if err != nil {
			logger.Errorf("failed to match rule (host: %s, path: %s): %s", hostname, path, err)

			// service not found
			return false, false, proxy.NewHTTPError(404, "Not Found")
		}

		// redirect: check path-level redirect first, then host-level redirect
		var redirectURL string
		var permanent bool
		var hasRedirect bool

		if pathBackend != nil && pathBackend.Redirect.URL != "" {
			redirectURL = pathBackend.Redirect.URL
			permanent = pathBackend.Redirect.Permanent
			hasRedirect = true
		} else if matchedRule.Backend.Redirect.URL != "" {
			redirectURL = matchedRule.Backend.Redirect.URL
			permanent = matchedRule.Backend.Redirect.Permanent
			hasRedirect = true
		}

		if hasRedirect {
			// If redirect URL is not a full URL (doesn't start with http:// or https://),
			// construct the full URL by keeping the original path and query parameters
			if !strings.HasPrefix(redirectURL, "http://") && !strings.HasPrefix(redirectURL, "https://") {
				// Use the same scheme as the original request
				scheme := "http"
				if ctx.Request.TLS != nil || ctx.Request.Header.Get("X-Forwarded-Proto") == "https" {
					scheme = "https"
				}

				// Build the redirect URL with original path and query
				redirectURL = fmt.Sprintf("%s://%s%s", scheme, redirectURL, path)
				if ctx.Request.URL.RawQuery != "" {
					redirectURL = fmt.Sprintf("%s?%s", redirectURL, ctx.Request.URL.RawQuery)
				}
			}

			if permanent {
				ctx.RedirectPermanent(redirectURL)
			} else {
				ctx.RedirectTemporary(redirectURL)
			}

			return false, true, nil
		}

		// handler: check path-level backend first, then host-level backend
		var handlerCfg *rule.Handler
		if pathBackend != nil && getBackendType(*pathBackend) == backendTypeHandler {
			handlerCfg = &pathBackend.Handler
		} else if getBackendType(matchedRule.Backend) == backendTypeHandler {
			handlerCfg = &matchedRule.Backend.Handler
		}

		if handlerCfg != nil {
			handlerType := handlerCfg.Type
			if handlerType == "" {
				handlerType = handlerTypeStaticResponse
			}

			switch handlerType {
			case handlerTypeStaticResponse:
				for k, v := range handlerCfg.Headers {
					ctx.Writer.Header().Set(k, v)
				}

				statusCode := int(handlerCfg.StatusCode)
				if statusCode == 0 {
					statusCode = http.StatusOK
				}
				ctx.Status(statusCode)
				ctx.Writer.Write([]byte(handlerCfg.Body))
			case handlerTypeFileServer:
				if handlerCfg.RootDir == "" {
					return false, false, proxy.NewHTTPError(http.StatusInternalServerError, "handler.root_dir is required for file_server")
				}

				indexFile := handlerCfg.IndexFile
				if indexFile == "" {
					indexFile = "index.html"
				}

				filePath := strings.TrimPrefix(pathlib.Clean(ctx.Path), "/")
				if filePath == "" || strings.HasSuffix(ctx.Path, "/") {
					filePath = indexFile
				}
				targetFilePath := filepath.Join(handlerCfg.RootDir, filepath.FromSlash(filePath))

				content, err := os.ReadFile(targetFilePath)
				if err != nil {
					return false, false, proxy.NewHTTPError(http.StatusNotFound, "Not Found")
				}

				if contentType := mime.TypeByExtension(filepath.Ext(targetFilePath)); contentType != "" {
					ctx.Writer.Header().Set("Content-Type", contentType)
				}
				ctx.Status(http.StatusOK)
				ctx.Writer.Write(content)
			case handlerTypeTemplates:
				if handlerCfg.RootDir == "" {
					return false, false, proxy.NewHTTPError(http.StatusInternalServerError, "handler.root_dir is required for templates")
				}

				templatePath := strings.TrimPrefix(pathlib.Clean(ctx.Path), "/")
				if templatePath == "" || strings.HasSuffix(ctx.Path, "/") {
					templatePath = "index.html"
				}
				targetTemplatePath := filepath.Join(handlerCfg.RootDir, filepath.FromSlash(templatePath))

				tpl, err := template.ParseFiles(targetTemplatePath)
				if err != nil {
					return false, false, proxy.NewHTTPError(http.StatusNotFound, "Not Found")
				}

				ctx.Status(http.StatusOK)
				if err := tpl.Execute(ctx.Writer, map[string]any{
					"Path":   ctx.Path,
					"Method": ctx.Method,
				}); err != nil {
					return false, false, proxy.NewHTTPError(http.StatusInternalServerError, "Template Render Failed")
				}
			case handlerTypeScript:
				if err := executeHandlerScript(ctx, handlerCfg); err != nil {
					return false, false, proxy.NewHTTPError(http.StatusInternalServerError, err.Error())
				}
				return false, true, nil
			default:
				return false, false, proxy.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("unsupported handler.type: %s", handlerType))
			}

			return false, true, nil
		}

		// If there's only redirect config but no service, skip validation
		if serviceIns.Name == "" {
			logger.Errorf("service name is empty for host: %s, path: %s", hostname, path)
			return false, false, proxy.NewHTTPError(500, "Service configuration is invalid")
		}

		// Validate client authentication before processing request
		if err := serviceIns.ValidateAuth(ctx.Request); err != nil {
			logger.Warnf("authentication failed for host: %s, path: %s: %s", hostname, path, err)

			// Set WWW-Authenticate header based on auth type
			if serviceIns.Auth.Type == "basic" {
				ctx.Writer.Header().Set("WWW-Authenticate", "Basic realm=\"Restricted\"")
			} else if serviceIns.Auth.Type == "bearer" {
				ctx.Writer.Header().Set("WWW-Authenticate", "Bearer")
			}

			ctx.Status(http.StatusUnauthorized)
			ctx.String(http.StatusUnauthorized, "Unauthorized")
			return false, true, nil
		}

		if err := serviceIns.Validate(); err != nil {
			return false, false, proxy.NewHTTPError(500, err.Error())
		}

		ips, err := serviceIns.CheckDNS()
		if err != nil {
			logger.Errorf("check dns error: %s", err)

			// exact service specify
			if matchedRule.HostType == "exact" {
				return false, false, proxy.NewHTTPError(503, "Service Unavailable")
			}

			// regular expression service specify, maybe the service is not found
			return false, false, proxy.NewHTTPError(404, "Service Not Found")
		}

		ctx.Logger.Debugf("[dns] service(%s) is ok (ips: %s)", serviceIns.Name, strings.Join(ips, ", "))

		// apply delay if configured
		if serviceIns.Request.Delay > 0 {
			delayDuration := time.Duration(serviceIns.Request.Delay) * time.Millisecond
			ctx.Logger.Debugf("[delay] applying delay of %v for service %s", delayDuration, serviceIns.Name)
			time.Sleep(delayDuration)
		}

		// apply timeout to the request context if configured
		if serviceIns.Request.Timeout > 0 {
			timeoutDuration := time.Duration(serviceIns.Request.Timeout) * time.Second
			ctx.Logger.Debugf("[timeout] setting timeout of %v for service %s", timeoutDuration, serviceIns.Name)
			timeoutCtx, cancel := context.WithTimeout(ctx.Request.Context(), timeoutDuration)
			_ = cancel // cancel will be called when request completes or context expires
			ctx.Request = ctx.Request.WithContext(timeoutCtx)
		}

		cfg.OnRequest = func(req, inReq *http.Request) error {
			req.URL.Scheme = serviceIns.Protocol
			req.URL.Host = serviceIns.Host()

			// apply host
			if serviceIns.Request.Host.Rewrite {
				req.Host = serviceIns.Host()
			}

			// apply path
			req.URL.Path = serviceIns.Rewrite(req.URL.Path)

			// apply headers
			for k, v := range serviceIns.Request.Headers {
				req.Header.Set(k, v)
			}

			// apply query
			if serviceIns.Request.Query != nil {
				originQuery := req.URL.Query()
				for k, v := range serviceIns.Request.Query {
					originQuery.Set(k, v)
				}
				req.URL.RawQuery = originQuery.Encode()
			}

			// apply timeout to the request context if configured
			if serviceIns.Request.Timeout > 0 {
				timeoutDuration := time.Duration(serviceIns.Request.Timeout) * time.Second
				timeoutCtx, cancel := context.WithTimeout(req.Context(), timeoutDuration)
				_ = cancel // cancel will be called when request completes or context expires
				req = req.WithContext(timeoutCtx)
			}

			// plugins
			for _, plugin := range c.plugins {
				if err := plugin.OnRequest(ctx, req); err != nil {
					return err
				}
			}

			return nil
		}

		cfg.OnResponse = func(res *http.Response, inReq *http.Request) error {
			for k, v := range serviceIns.Response.Headers {
				ctx.Writer.Header().Set(k, v)
			}

			// plugins
			for _, plugin := range c.plugins {
				if err := plugin.OnResponse(ctx, res); err != nil {
					return err
				}
			}

			ctx.Writer.Header().Del("X-Powered-By")
			ctx.Writer.Header().Set("X-Powered-By", fmt.Sprintf("gozoox-ingress/%s", c.version))
			return nil
		}

		ctx.Logger.Infof("[host: %s, target: %s] \"%s %s %s\" %d", hostname, serviceIns.Target(), method, path, ctx.Request.Proto, ctx.StatusCode())

		return
	}))

	return nil
}
