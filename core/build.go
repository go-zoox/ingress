package core

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"mime"
	"net/http"
	"os"
	pathlib "path"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/ingress/core/ratelimit"
	"github.com/go-zoox/ingress/core/security"
	"github.com/go-zoox/ingress/core/service"
	"github.com/go-zoox/ingress/core/waf"
	"github.com/go-zoox/proxy"
	"github.com/go-zoox/zoox"
	"github.com/go-zoox/zoox/middleware"
)

func (c *core) build() error {
	// config
	c.app.Config.Port = int(c.cfg.Port)
	c.app.Config.HTTPSPort = int(c.cfg.HTTPS.Port)
	c.app.Config.EnableH2C = c.cfg.EnableH2C
	c.app.Config.EnableHTTP3 = c.cfg.HTTPS.EnableHTTP3
	c.app.Config.HTTP3Port = int(c.cfg.HTTPS.HTTP3Port)
	c.app.Config.HTTP3AltSvcMaxAge = int(c.cfg.HTTPS.HTTP3AltSvcMaxAge)

	// middlewares
	c.app.Use(func(ctx *zoox.Context) {
		if ctx.Path == c.ingressStatusPath {
			c.writeIngressStatus(ctx)
			return
		}

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
	c.app.Use(c.ingressProxy(func(ctx *zoox.Context, cfg *middleware.ProxyConfig) (next, stop bool, err error) {
		reqStart := time.Now()
		hostname := ctx.Hostname()
		method := ctx.Method
		path := ctx.Path
		rawQuery := ctx.Request.URL.RawQuery
		pageDetail := ErrorPageDetail{Hostname: hostname, Path: path, Method: method}
		c.fillProxyErrorPages(cfg, ctx.Request, pageDetail)

		if shouldRedirectFromHTTP(ctx.Request, path, c.cfg) {
			redirectURL := buildHTTPSRedirectURL(hostname, path, rawQuery, c.cfg.HTTPS.Port)
			rf := c.cfg.HTTPS.RedirectFromHTTP
			applyRedirect(ctx, redirectURL, rf.Permanent, rf.WithOriginMethodAndBody)
			return false, true, nil
		}

		serviceIns, matchedRule, pathBackend, hostSm, pathSm, ruleIdx, pathIdx, err := c.match(ctx, hostname, path)
		if err != nil {
			c.app.Logger().Warnf("no route matched (host: %s, path: %s): %s", hostname, path, err)
			c.writeErrorPage(ctx, http.StatusNotFound, c.securityGlobal(), ErrorPageDetail{
				Hostname: hostname,
				Path:     path,
				Method:   method,
				Reason:   matchErrorReason(err),
			})
			return false, true, nil
		}

		secProf := c.securityForMatch(ruleIdx, pathIdx)
		secTarget := "-"
		if serviceIns != nil {
			secTarget = serviceIns.Target()
		}
		if c.handleSecurityPreflight(ctx, secProf, hostname, secTarget, method, path, ctx.Request.Proto, reqStart) {
			return false, true, nil
		}

		if c.rateLimits != nil {
			var ruleRL *ratelimit.Policy
			if ruleIdx >= 0 && ruleIdx < len(c.rateLimits.ByRule) {
				ruleRL = c.rateLimits.ByRule[ruleIdx]
			}
			if blocked, retryAfter := ratelimit.Check(ctx.Request, c.rateLimits.Global, ruleRL, ruleIdx); blocked {
				ctx.SetHeader("Retry-After", ratelimit.ParseRetryAfter(retryAfter))
				applySecurityHeaders(ctx, secProf)
				ctx.Status(http.StatusTooManyRequests)
				ctx.String(http.StatusTooManyRequests, "Too Many Requests")
				target := "-"
				if serviceIns != nil {
					target = serviceIns.Target()
				}
				ctx.Logger.Infof("%s", formatAccessLog(ctx.Request, hostname, target, method, path, ctx.Request.Proto, http.StatusTooManyRequests, time.Since(reqStart), accessLogMeta{
					RateLimitBlock:         true,
					UpstreamStatus:         http.StatusTooManyRequests,
					UpstreamResponseLength: -1,
				}))
				return false, true, nil
			}
		}

		wafProf := c.wafFallback
		if ruleIdx >= 0 && ruleIdx < len(c.wafByRuleIdx) {
			wafProf = c.wafByRuleIdx[ruleIdx]
		}
		if wafProf != nil && c.IsWAFEnabled() && waf.CheckRequest(wafProf, ctx.Request, hostname, path, method, func(action, rule, clientIP string) {
			if c.wafCallback != nil {
				ua := ""
				if ctx.Request != nil {
					ua = ctx.Request.UserAgent()
				}
				c.wafCallback.OnWAFEvent(action, rule, hostname, path, clientIP, ua)
			}
		}) {
			if shouldUseWAFErrorPage(wafProf.BlockStatus, wafProf.BlockContentType, wafProf.BlockBody) {
				c.writeErrorPage(ctx, http.StatusForbidden, secProf, pageDetail)
			} else {
				ctx.SetHeader("Content-Type", wafProf.BlockContentType)
				applySecurityHeaders(ctx, secProf)
				ctx.String(wafProf.BlockStatus, wafProf.BlockBody)
			}
			target := "-"
			if serviceIns != nil {
				target = serviceIns.Target()
			}
			ctx.Logger.Infof("%s", formatAccessLog(ctx.Request, hostname, target, method, path, ctx.Request.Proto, wafProf.BlockStatus, time.Since(reqStart), accessLogMeta{
				WAFBlock:               true,
				UpstreamStatus:         wafProf.BlockStatus,
				UpstreamResponseLength: -1,
			}))
			return false, true, nil
		}

		if block, settings := c.maintenanceDecision(ruleIdx, hostname, path, ctx.Request); block {
			c.writeMaintenanceResponse(ctx, secProf, settings, pageDetail)
			target := "-"
			if serviceIns != nil {
				target = serviceIns.Target()
			}
			ctx.Logger.Infof("%s", formatAccessLog(ctx.Request, hostname, target, method, path, ctx.Request.Proto, http.StatusServiceUnavailable, time.Since(reqStart), accessLogMeta{
				MaintenanceBlock:       true,
				UpstreamStatus:         http.StatusServiceUnavailable,
				UpstreamResponseLength: -1,
			}))
			return false, true, nil
		}

		// After route resolution: apply backend.redirect when Redirect.URL is set (path backend overrides rule backend).
		// Redirect-only configs keep Backend.Type as default "service" with backend.redirect only; otherwise matched upstream proxy continues below.
		// Next block handles Backend.Type "handler".
		var redirectURL string
		var permanent bool
		var withOriginMethodAndBody bool
		var hasRedirect bool

		if pathBackend != nil && pathBackend.Redirect.URL != "" {
			redirectURL = pathBackend.Redirect.URL
			permanent = pathBackend.Redirect.Permanent
			withOriginMethodAndBody = pathBackend.Redirect.WithOriginMethodAndBody
			hasRedirect = true
		} else if matchedRule.Backend.Redirect.URL != "" {
			redirectURL = matchedRule.Backend.Redirect.URL
			permanent = matchedRule.Backend.Redirect.Permanent
			withOriginMethodAndBody = matchedRule.Backend.Redirect.WithOriginMethodAndBody
			hasRedirect = true
		}

		if hasRedirect {
			// Optional HTTP cache: same fingerprint as handler/service; store GET redirects after URL expansion (see http_cache.go).
			rb := effectiveRouteBackend(matchedRule, pathBackend)
			policyRedirect := httpCachePolicyForRequest(path, normalizeHTTPCache(rb.Cache))
			var redirectCacheKey string
			redirectCacheStart := time.Now()
			var mayStoreRedirect bool
			if policyRedirect != nil && httpCacheMethodAllowed(method, policyRedirect) && !httpCacheRequestBypasses(ctx.Request, policyRedirect) {
				if prepared, skip := httpCachePrepareRequest(ctx, policyRedirect); skip {
					policyRedirect = nil
				} else {
					policyRedirect = prepared
				}
			}
			if policyRedirect != nil && httpCacheMethodAllowed(method, policyRedirect) && !httpCacheRequestBypasses(ctx.Request, policyRedirect) {
				redirectCacheKey = buildHTTPCacheStorageKey(ctx.Request, hostname, path, policyRedirect)
				applySecurityHeaders(ctx, secProf)
				if hit, code, ulen := tryServeHTTPCache(ctx, policyRedirect, redirectCacheKey); hit {
					rdDur := time.Since(redirectCacheStart)
					ctx.Logger.Infof("%s", formatAccessLog(ctx.Request, hostname, "redirect", method, path, ctx.Request.Proto, code, rdDur, accessLogMeta{
						CacheHit:               true,
						UpstreamStatus:         code,
						UpstreamResponseLength: ulen,
						UpstreamResponseTime:   rdDur,
					}))
					return false, true, nil
				}
				mayStoreRedirect = method == http.MethodGet
			}

			redirectURL = expandRedirectURL(matchedRule, hostname, redirectURL, hostSm, pathSm)
			// If redirect URL is not a full URL (doesn't start with http:// or https://),
			// construct the full URL by keeping the original path and query parameters
			if !strings.HasPrefix(redirectURL, "http://") && !strings.HasPrefix(redirectURL, "https://") {
				// Use the same scheme as the original request
				scheme := schemeHTTP
				if ctx.Request.TLS != nil || strings.EqualFold(ctx.Request.Header.Get(headerXForwardedProto), schemeHTTPS) {
					scheme = schemeHTTPS
				}

				// Build the redirect URL with original path and query
				redirectURL = fmt.Sprintf("%s://%s%s", scheme, redirectURL, path)
				if ctx.Request.URL.RawQuery != "" {
					redirectURL = fmt.Sprintf("%s?%s", redirectURL, ctx.Request.URL.RawQuery)
				}
			}

			if mayStoreRedirect && policyRedirect != nil && redirectCacheKey != "" {
				code := redirectStatusFromFlags(permanent, withOriginMethodAndBody)
				if httpCacheShouldStoreRedirect(code, redirectURL) {
					h := http.Header{}
					h.Set("Location", redirectURL)
					ttl := httpCacheTTLFromResponseHeader(h, policyRedirect.TTL)
					ent := &httpCacheEntry{
						StatusCode: code,
						Header:     map[string][]string{"Location": {redirectURL}},
						Body:       nil,
					}
					_ = ctx.Cache().Set(redirectCacheKey, ent, ttl)
				}
			}

			applySecurityHeaders(ctx, secProf)
			applyRedirect(ctx, redirectURL, permanent, withOriginMethodAndBody)
			rdCode := redirectStatusFromFlags(permanent, withOriginMethodAndBody)
			ctx.Logger.Infof("%s", formatAccessLog(ctx.Request, hostname, "redirect", method, path, ctx.Request.Proto, rdCode, time.Since(redirectCacheStart), accessLogMeta{
				UpstreamStatus:         rdCode,
				UpstreamResponseLength: -1,
			}))

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
			// Optional HTTP cache for handler backends: try GET/HEAD read; on GET miss capture body via zooxHTTPCacheCaptureRW for later Set.
			origWriter := ctx.Writer
			handlerType := handlerCfg.Type
			if handlerType == "" {
				handlerType = handlerTypeStaticResponse
			}

			rb := effectiveRouteBackend(matchedRule, pathBackend)
			policyHandler := httpCachePolicyForRequest(path, normalizeHTTPCache(rb.Cache))
			var handlerCacheKey string
			var captureBuf *bytes.Buffer
			handlerCacheStart := time.Now()
			if policyHandler != nil && httpCacheMethodAllowed(method, policyHandler) && !httpCacheRequestBypasses(ctx.Request, policyHandler) {
				if prepared, skip := httpCachePrepareRequest(ctx, policyHandler); skip {
					policyHandler = nil
				} else {
					policyHandler = prepared
				}
			}
			if policyHandler != nil && httpCacheMethodAllowed(method, policyHandler) && !httpCacheRequestBypasses(ctx.Request, policyHandler) {
				handlerCacheKey = buildHTTPCacheStorageKey(ctx.Request, hostname, path, policyHandler)
				applySecurityHeaders(ctx, secProf)
				if hit, code, ulen := tryServeHTTPCache(ctx, policyHandler, handlerCacheKey); hit {
					hDur := time.Since(handlerCacheStart)
					ctx.Logger.Infof("%s", formatAccessLog(ctx.Request, hostname, handlerAccessTarget(handlerType), method, path, ctx.Request.Proto, code, hDur, accessLogMeta{
						CacheHit:               true,
						UpstreamStatus:         code,
						UpstreamResponseLength: ulen,
						UpstreamResponseTime:   hDur,
					}))
					return false, true, nil
				}
				if httpCacheShouldCaptureHandlerResponse(method, policyHandler) {
					captureBuf = &bytes.Buffer{}
					ctx.Writer = &zooxHTTPCacheCaptureRW{ResponseWriter: origWriter, buf: captureBuf}
				}
			}
			defer func() {
				if captureBuf != nil {
					ctx.Writer = origWriter
				}
			}()

			handlerStart := time.Now()
			applySecurityHeaders(ctx, secProf)

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
			default:
				return false, false, proxy.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("unsupported handler.type: %s", handlerType))
			}

			if captureBuf != nil && policyHandler != nil && handlerCacheKey != "" {
				// Persist handler output only when policy and response headers allow (200, no Vary / no-store / Set-Cookie, size cap).
				st := ctx.Writer.Status()
				hdr := ctx.Writer.Header()
				ttl := httpCacheTTLFromResponseHeader(hdr, policyHandler.TTL)
				if httpCacheShouldStoreHandler(st, hdr, captureBuf.Len(), policyHandler) {
					ent := &httpCacheEntry{
						StatusCode: st,
						Header:     cloneHeadersForCache(hdr, policyHandler.SkipVary),
						Body:       append([]byte(nil), captureBuf.Bytes()...),
					}
					_ = ctx.Cache().Set(handlerCacheKey, ent, ttl)
				}
			}

			handlerStatusCode := ctx.Writer.Status()
			handlerDuration := time.Since(handlerStart)
			var respLen int64 = -1
			if captureBuf != nil {
				respLen = int64(captureBuf.Len())
			}
			ctx.Logger.Infof("%s", formatAccessLog(ctx.Request, hostname, handlerAccessTarget(handlerType), method, path, ctx.Request.Proto, handlerStatusCode, handlerDuration, accessLogMeta{
				UpstreamStatus:         handlerStatusCode,
				UpstreamResponseLength: respLen,
				UpstreamResponseTime:   handlerDuration,
			}))

			return false, true, nil
		}

		// If there's only redirect config but no service, skip validation
		if serviceIns.Name == "" {
			c.app.Logger().Errorf("service name is empty for host: %s, path: %s", hostname, path)
			return false, false, proxy.NewHTTPError(500, "Service configuration is invalid")
		}

		// Handle OAuth2 / OIDC session authentication (redirect flow).
		if serviceIns.Auth.IsEnabled() {
			switch serviceIns.Auth.Type {
			case authTypeOAuth2:
				redirected, err := serviceIns.ValidateOAuth2(ctx)
				if err != nil {
					c.app.Logger().Warnf("oauth2 authentication failed for host: %s: %s", hostname, err)
					setUnauthorizedChallenge(ctx, serviceIns)
					c.writeErrorPage(ctx, http.StatusUnauthorized, secProf, pageDetail)
					return false, true, nil
				}
				if redirected {
					return false, true, nil
				}
				if serviceIns.Auth.OAuth2.Connect.Enabled {
					if err := serviceIns.InjectConnectHeaders(ctx); err != nil {
						c.app.Logger().Warnf("failed to inject connect headers: %v", err)
					}
				}
			case authTypeOIDC:
				if strings.TrimSpace(serviceIns.Auth.OIDC.Provider) != "" {
					redirected, err := serviceIns.ValidateOIDCSession(ctx)
					if err != nil {
						c.app.Logger().Warnf("oidc authentication failed for host: %s: %s", hostname, err)
						setUnauthorizedChallenge(ctx, serviceIns)
						c.writeErrorPage(ctx, http.StatusUnauthorized, secProf, pageDetail)
						return false, true, nil
					}
					if redirected {
						return false, true, nil
					}
				}
			}
		}

		// Validate client authentication before processing request
		if err := serviceIns.ValidateAuth(ctx.Request); err != nil {
			c.app.Logger().Warnf("authentication failed for host: %s, path: %s: %s", hostname, path, err)
			setUnauthorizedChallenge(ctx, serviceIns)
			c.writeErrorPage(ctx, http.StatusUnauthorized, secProf, pageDetail)
			return false, true, nil
		}

		if err := serviceIns.Validate(); err != nil {
			return false, false, proxy.NewHTTPError(500, err.Error())
		}

		ips, err := serviceIns.CheckDNS()
		if err != nil {
			c.app.Logger().Errorf("check dns error (service=%s host=%s path=%s): %s", serviceIns.Name, hostname, path, err)

			// exact service specify
			if matchedRule.HostType == hostTypeExact {
				c.writeErrorPage(ctx, http.StatusServiceUnavailable, secProf, ErrorPageDetail{
					Hostname: hostname,
					Path:     path,
					Method:   method,
					Reason:   err.Error(),
				})
				return false, true, nil
			}

			// regular expression service specify, maybe the service is not found
			c.writeErrorPage(ctx, http.StatusNotFound, secProf, ErrorPageDetail{
				Hostname: hostname,
				Path:     path,
				Method:   method,
				Reason:   fmt.Sprintf("Service: %s · %s", serviceIns.Name, err.Error()),
			})
			return false, true, nil
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

		proxyStart := time.Now()

		// Service HTTP cache: read before RoundTrip; write in OnResponse after buffering (GET upstream responses only for populate).
		routeBackend := effectiveRouteBackend(matchedRule, pathBackend)
		pc := httpCachePolicyForRequest(path, normalizeHTTPCache(routeBackend.Cache))
		var httpCacheStoreKey string
		var httpCacheMayStore bool
		if pc != nil {
			if httpCacheMethodAllowed(method, pc) && !httpCacheRequestBypasses(ctx.Request, pc) {
				if prepared, skip := httpCachePrepareRequest(ctx, pc); skip {
					pc = nil
				} else {
					pc = prepared
				}
			}
			if pc != nil && httpCacheMethodAllowed(method, pc) && !httpCacheRequestBypasses(ctx.Request, pc) {
				httpCacheStoreKey = buildHTTPCacheStorageKey(ctx.Request, hostname, path, pc)
				applySecurityHeaders(ctx, secProf)
				if hit, code, ulen := tryServeHTTPCache(ctx, pc, httpCacheStoreKey); hit {
					hitDur := time.Since(proxyStart)
					ctx.Logger.Infof("%s", formatAccessLog(ctx.Request, hostname, serviceIns.Target(), method, path, ctx.Request.Proto, code, hitDur, accessLogMeta{
						CacheHit:               true,
						UpstreamStatus:         code,
						UpstreamResponseLength: ulen,
						UpstreamResponseTime:   hitDur,
					}))
					return false, true, nil
				}
				httpCacheMayStore = true
			}
		}

		hostRewrite := effectiveHostRewrite(serviceIns, pathBackend, matchedRule)

		cfg.OnRequest = func(req, inReq *http.Request) error {
			req.URL.Scheme = serviceIns.Protocol
			req.URL.Host = serviceIns.Host()

			// apply host
			if hostRewrite {
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
				res.Header.Set(k, v)
			}

			if secProf != nil && secProf.Active {
				security.ApplyHeaders(res.Header, secProf, inReq)
			}

			// plugins
			for _, plugin := range c.plugins {
				if err := plugin.OnResponse(ctx, res); err != nil {
					return err
				}
			}

			res.Header.Set("X-Powered-By", fmt.Sprintf("gozoox-ingress/%s", c.version))

			// Service HTTP cache write: headers are final after plugins; allowed methods may extend the shared ctx.Cache.
			if pc != nil && httpCacheMayStore && httpCacheStoreKey != "" && httpCacheMethodAllowed(inReq.Method, pc) {
				body, err := io.ReadAll(res.Body)
				_ = res.Body.Close()
				if err != nil {
					return err
				}
				if httpCacheShouldStore(res, len(body), pc) {
					ttl := httpCacheResponseTTL(res, pc.TTL)
					ent := &httpCacheEntry{
						StatusCode: res.StatusCode,
						Header:     cloneHeadersForCache(res.Header, pc.SkipVary),
						Body:       append([]byte(nil), body...),
					}
					_ = ctx.Cache().Set(httpCacheStoreKey, ent, ttl)
				}
				res.Body = io.NopCloser(bytes.NewReader(body))
			}

			upstreamDuration := time.Since(proxyStart)
			ctx.Logger.Infof("%s", formatAccessLog(ctx.Request, hostname, serviceIns.Target(), method, path, ctx.Request.Proto, res.StatusCode, upstreamDuration, accessLogMeta{
				UpstreamStatus:         res.StatusCode,
				UpstreamResponseLength: res.ContentLength,
				UpstreamResponseTime:   upstreamDuration,
			}))

			return nil
		}

		return
	}))

	return nil
}

func applyRedirect(ctx *zoox.Context, url string, permanent, withOriginMethodAndBody bool) {
	if withOriginMethodAndBody {
		if permanent {
			ctx.RedirectPermanentWithOriginMethodAndBody(url)
		} else {
			ctx.RedirectTemporaryWithOriginMethodAndBody(url)
		}
		return
	}
	if permanent {
		ctx.RedirectPermanent(url)
	} else {
		ctx.RedirectTemporary(url)
	}
}

func shouldRedirectFromHTTP(req *http.Request, path string, cfg *Config) bool {
	// Only enable forced redirect when HTTPS listener is configured.
	if cfg.HTTPS.Port == 0 {
		return false
	}

	if !cfg.HTTPS.RedirectFromHTTP.Enabled {
		return false
	}

	if req.TLS != nil {
		return false
	}

	if strings.EqualFold(req.Header.Get(headerXForwardedProto), schemeHTTPS) {
		return false
	}

	for _, excludedPath := range cfg.HTTPS.RedirectFromHTTP.ExcludePaths {
		if path == excludedPath {
			return false
		}
	}

	return true
}

func buildHTTPSRedirectURL(hostname, path, rawQuery string, httpsPort int64) string {
	host := hostname
	if httpsPort != 0 && httpsPort != 443 {
		host = fmt.Sprintf("%s:%d", host, httpsPort)
	}

	redirectURL := fmt.Sprintf("https://%s%s", host, path)
	if rawQuery != "" {
		redirectURL = fmt.Sprintf("%s?%s", redirectURL, rawQuery)
	}

	return redirectURL
}

func setUnauthorizedChallenge(ctx *zoox.Context, serviceIns *service.Service) {
	if serviceIns == nil {
		return
	}
	switch serviceIns.Auth.Type {
	case authTypeBasic:
		ctx.Writer.Header().Set(headerWWWAuthenticate, authChallengeBasic)
	case authTypeBearer, authTypeJWT, authTypeOIDC:
		ctx.Writer.Header().Set(headerWWWAuthenticate, authChallengeBearer)
	}
}
