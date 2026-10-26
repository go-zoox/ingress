package core

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-zoox/ingress/core/ratelimit"
	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/ingress/core/service"
	"github.com/go-zoox/zoox"
)

const (
	defaultConnectProxyConnectTimeout  = 10 * time.Second
	defaultConnectProxyResponseTimeout = 10 * time.Second
	defaultConnectProxyIdleTimeout     = 10 * time.Minute
)

// compiledConnectProxyRule is matched by TLS SNI. CONNECT/absolute-form requests
// carry the destination in Host, so the ordinary router must not be used here.
type compiledConnectProxyRule struct {
	entry             compiledRuleEntry
	targetPatterns    []*regexp.Regexp
	connectTimeout    time.Duration
	responseTimeout   time.Duration
	idleTimeout       time.Duration
	maxConnections    int64
	activeConnections atomic.Int64
}

func compileConnectProxyRules(cfg *Config) ([]compiledConnectProxyRule, error) {
	compiled := make([]compiledConnectProxyRule, 0)

	for i := range cfg.Rules {
		r := &cfg.Rules[i]
		if !r.IsEnabled() || !r.ConnectProxy.Enabled {
			continue
		}
		if r.ConnectProxy.ConnectTimeout < 0 || r.ConnectProxy.ResponseTimeout < 0 || r.ConnectProxy.IdleTimeout < 0 || r.ConnectProxy.MaxConnections < 0 {
			return nil, fmt.Errorf("rules[%d].connect_proxy: timeout and max_connections must not be negative", i)
		}
		if r.ConnectProxy.Upstream.Name != "" {
			if err := r.ConnectProxy.Upstream.Validate(); err != nil {
				return nil, fmt.Errorf("rules[%d].connect_proxy.upstream: %w", i, err)
			}
			protocol := strings.ToLower(strings.TrimSpace(r.ConnectProxy.Upstream.Protocol))
			if protocol != "" && protocol != "http" && protocol != "https" {
				return nil, fmt.Errorf("rules[%d].connect_proxy.upstream.protocol: unsupported protocol %q", i, r.ConnectProxy.Upstream.Protocol)
			}
		} else if getBackendType(r.Backend) != backendTypeService {
			return nil, fmt.Errorf("rules[%d].connect_proxy: upstream or a service backend is required", i)
		}

		patterns := make([]*regexp.Regexp, 0, len(r.ConnectProxy.AllowedTargets))
		for j, pattern := range r.ConnectProxy.AllowedTargets {
			re, err := regexp.Compile(pattern)
			if err != nil {
				return nil, fmt.Errorf("rules[%d].connect_proxy.allowed_targets[%d]: %w", i, j, err)
			}
			patterns = append(patterns, re)
		}

		ht := effectiveHostType(r.HostType, r.Host)
		r.HostType = ht
		entry := compiledRuleEntry{hostType: ht, ruleIndex: i}
		switch ht {
		case hostTypeExact:
			entry.exactHost = strings.ToLower(r.Host)
		case hostTypeRegex:
			re, err := regexp.Compile(r.Host)
			if err != nil {
				return nil, fmt.Errorf("rules[%d].connect_proxy host regex: %w", i, err)
			}
			entry.re = re
		case hostTypeWildcard:
			pat := wildCardToRegexp(r.Host)
			re, err := regexp.Compile(pat)
			if err != nil {
				return nil, fmt.Errorf("rules[%d].connect_proxy host wildcard: %w", i, err)
			}
			entry.re = re
			entry.wildcardRewriterFrom = pat
		default:
			return nil, fmt.Errorf("rules[%d].connect_proxy: unsupported host type %q", i, ht)
		}

		compiled = append(compiled, compiledConnectProxyRule{
			entry:           entry,
			targetPatterns:  patterns,
			connectTimeout:  secondsOrDefault(r.ConnectProxy.ConnectTimeout, defaultConnectProxyConnectTimeout),
			responseTimeout: secondsOrDefault(r.ConnectProxy.ResponseTimeout, defaultConnectProxyResponseTimeout),
			idleTimeout:     secondsOrDefault(r.ConnectProxy.IdleTimeout, defaultConnectProxyIdleTimeout),
			maxConnections:  r.ConnectProxy.MaxConnections,
		})
	}

	return compiled, nil
}

func (c *core) matchConnectProxy(sni string) (*service.Service, *rule.Rule, *compiledConnectProxyRule, error) {
	for i := range c.connectProxyRules {
		compiled := &c.connectProxyRules[i]
		submatches, matched := matchCompiledHostEntry(compiled.entry, sni)
		if !matched {
			continue
		}

		ruleCfg := &c.cfg.Rules[compiled.entry.ruleIndex]
		matcher, err := hostMatcherFromMatchedRule(
			ruleCfg,
			sni,
			compiled.entry.wildcardRewriterFrom,
			submatches,
			compiled.entry.ruleIndex,
		)
		if err != nil {
			return nil, nil, nil, err
		}

		return matcher.Service, ruleCfg, compiled, nil
	}

	return nil, nil, nil, errors.New("no CONNECT proxy route matched TLS SNI")
}

func matchCompiledHostEntry(entry compiledRuleEntry, host string) ([]string, bool) {
	switch entry.hostType {
	case hostTypeExact:
		return nil, entry.exactHost == strings.ToLower(host)
	case hostTypeRegex:
		submatches := entry.re.FindStringSubmatch(host)
		return submatches, submatches != nil
	case hostTypeWildcard:
		return nil, entry.re.MatchString(host)
	default:
		return nil, false
	}
}

func connectProxyUpstream(ruleCfg *rule.Rule, fallback *service.Service) (*service.Service, error) {
	if ruleCfg.ConnectProxy.Upstream.Name != "" {
		upstream := ruleCfg.ConnectProxy.Upstream
		if err := upstream.Validate(); err != nil {
			return nil, err
		}
		return &upstream, nil
	}

	if fallback == nil {
		return nil, errors.New("CONNECT proxy upstream is not configured")
	}
	return fallback, nil
}

func (r *compiledConnectProxyRule) acquire() bool {
	if r.maxConnections <= 0 {
		return true
	}
	if r.activeConnections.Add(1) > r.maxConnections {
		r.activeConnections.Add(-1)
		return false
	}
	return true
}

func (r *compiledConnectProxyRule) release() {
	if r.maxConnections > 0 {
		r.activeConnections.Add(-1)
	}
}

// handleConnectProxy handles CONNECT after ingress terminates TLS. The client
// connection is hijacked only after the upstream proxy accepted CONNECT.
func (c *core) handleConnectProxy(ctx *zoox.Context) (bool, bool, error) {
	req := ctx.Request
	started := time.Now()

	if req.TLS == nil {
		return c.failConnect(ctx, http.StatusForbidden, "CONNECT requires TLS", nil)
	}
	sni := strings.ToLower(req.TLS.ServerName)
	if sni == "" {
		return c.failConnect(ctx, http.StatusMisdirectedRequest, "CONNECT requires TLS SNI", nil)
	}
	if req.ProtoMajor != 1 {
		return c.failConnect(ctx, http.StatusHTTPVersionNotSupported, "CONNECT requires HTTP/1.1", nil)
	}

	target := connectTarget(req)
	if target == "" {
		return c.failConnect(ctx, http.StatusBadRequest, "CONNECT target is missing", nil)
	}

	fallback, ruleCfg, compiled, err := c.matchConnectProxy(sni)
	if err != nil {
		c.app.Logger().Warnf("[connect-proxy] no route matched (sni=%s, target=%s): %s", sni, target, err)
		return c.failConnect(ctx, http.StatusNotFound, "No CONNECT route", err)
	}
	if c.proxyRateLimited(ctx, compiled.entry.ruleIndex) {
		return false, true, nil
	}
	if !targetAllowed(compiled.targetPatterns, strings.ToLower(target)) {
		return c.failConnect(ctx, http.StatusForbidden, "CONNECT target is not allowed", nil)
	}
	if !compiled.acquire() {
		return c.failConnect(ctx, http.StatusServiceUnavailable, "Too many CONNECT connections", nil)
	}
	defer compiled.release()

	upstream, err := connectProxyUpstream(ruleCfg, fallback)
	if err != nil {
		return c.failConnect(ctx, http.StatusInternalServerError, "CONNECT route is invalid", err)
	}
	backendURL, err := url.Parse(upstream.Target())
	if err != nil {
		return c.failConnect(ctx, http.StatusInternalServerError, "CONNECT route upstream is invalid", err)
	}

	dialer := &net.Dialer{Timeout: compiled.connectTimeout, KeepAlive: 30 * time.Second}
	var upstreamConn net.Conn
	if backendURL.Scheme == "https" {
		host, _, _ := net.SplitHostPort(backendURL.Host)
		upstreamConn, err = tls.DialWithDialer(dialer, "tcp", backendURL.Host, &tls.Config{ServerName: host})
	} else {
		upstreamConn, err = dialer.Dial("tcp", backendURL.Host)
	}
	if err != nil {
		c.app.Logger().Warnf("[connect-proxy] failed to connect upstream %s: %s", backendURL.Host, err)
		return c.failConnect(ctx, http.StatusBadGateway, "Failed to connect proxy upstream", err)
	}
	defer upstreamConn.Close()

	upstreamReq := buildUpstreamConnect(req, target)
	if err := writeConnectRequestWithDeadline(upstreamConn, upstreamReq, compiled.connectTimeout); err != nil {
		c.app.Logger().Warnf("[connect-proxy] failed to write CONNECT to upstream: %s", err)
		return c.failConnect(ctx, http.StatusBadGateway, "Failed to forward CONNECT", err)
	}

	backendReader := bufio.NewReader(upstreamConn)
	if compiled.responseTimeout > 0 {
		_ = upstreamConn.SetReadDeadline(time.Now().Add(compiled.responseTimeout))
	}
	response, err := http.ReadResponse(backendReader, upstreamReq)
	_ = upstreamConn.SetReadDeadline(time.Time{})
	if err != nil {
		c.app.Logger().Warnf("[connect-proxy] failed to read CONNECT response: %s", err)
		return c.failConnect(ctx, http.StatusBadGateway, "Invalid CONNECT response from proxy upstream", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode > 299 {
		writeProxyResponse(ctx, response)
		elapsed := time.Since(started)
		c.logAccess(ctx, sni, backendURL.Host, http.MethodConnect, target, req.Proto, response.StatusCode, elapsed, accessLogMeta{
			UpstreamStatus:         response.StatusCode,
			UpstreamResponseLength: response.ContentLength,
			UpstreamResponseTime:   elapsed,
		})
		return false, true, nil
	}

	hijacker, ok := ctx.Writer.(http.Hijacker)
	if !ok {
		return c.failConnect(ctx, http.StatusInternalServerError, "CONNECT is not supported by this response writer", nil)
	}
	clientConn, clientRW, err := hijacker.Hijack()
	if err != nil {
		return c.failConnect(ctx, http.StatusInternalServerError, "Failed to hijack CONNECT connection", err)
	}
	if _, err := clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
		_ = clientConn.Close()
		return false, true, nil
	}

	clientTunnel := wrapIdleTimeout(clientConn, compiled.idleTimeout)
	upstreamTunnel := wrapIdleTimeout(upstreamConn, compiled.idleTimeout)
	clientReader := wrapIdleTimeoutReader(clientConn, clientRW.Reader, compiled.idleTimeout)
	upstreamReader := wrapIdleTimeoutReader(upstreamConn, backendReader, compiled.idleTimeout)
	c.app.Logger().Debugf("[connect-proxy] %s (sni=%s) established to %s", target, sni, backendURL.Host)
	tunnelConnectPair(clientTunnel, clientReader, upstreamTunnel, upstreamReader)
	_ = clientConn.Close()
	_ = upstreamConn.Close()

	elapsed := time.Since(started)
	c.logAccess(ctx, sni, backendURL.Host, http.MethodConnect, target, req.Proto, http.StatusOK, elapsed, accessLogMeta{
		UpstreamStatus:         response.StatusCode,
		UpstreamResponseLength: -1,
		UpstreamResponseTime:   elapsed,
	})
	return false, true, nil
}

// handleHTTPProxy forwards an absolute-form HTTP proxy request sent to the HTTPS
// proxy listener. Unlike CONNECT it stays an ordinary HTTP request and is relayed
// through a proxy-configured http.Transport.
func (c *core) handleHTTPProxy(ctx *zoox.Context) (bool, bool, error) {
	req := ctx.Request
	started := time.Now()

	if req.TLS == nil {
		return c.failConnect(ctx, http.StatusForbidden, "Proxy request requires TLS", nil)
	}
	sni := strings.ToLower(req.TLS.ServerName)
	if sni == "" {
		return c.failConnect(ctx, http.StatusMisdirectedRequest, "Proxy request requires TLS SNI", nil)
	}
	if req.URL == nil || !req.URL.IsAbs() || req.URL.Hostname() == "" {
		return c.failConnect(ctx, http.StatusBadRequest, "Proxy request target is missing", nil)
	}
	if req.URL.Scheme != "http" {
		return c.failConnect(ctx, http.StatusBadRequest, "HTTPS proxy targets must use CONNECT", nil)
	}

	fallback, ruleCfg, compiled, err := c.matchConnectProxy(sni)
	if err != nil {
		return c.failConnect(ctx, http.StatusNotFound, "No proxy route", err)
	}
	if c.proxyRateLimited(ctx, compiled.entry.ruleIndex) {
		return false, true, nil
	}

	target := proxyRequestAuthority(req)
	if !targetAllowed(compiled.targetPatterns, target) {
		return c.failConnect(ctx, http.StatusForbidden, "Proxy target is not allowed", nil)
	}
	if !compiled.acquire() {
		return c.failConnect(ctx, http.StatusServiceUnavailable, "Too many proxy connections", nil)
	}
	defer compiled.release()

	upstream, err := connectProxyUpstream(ruleCfg, fallback)
	if err != nil {
		return c.failConnect(ctx, http.StatusInternalServerError, "Proxy route is invalid", err)
	}
	backendURL, err := url.Parse(upstream.Target())
	if err != nil {
		return c.failConnect(ctx, http.StatusInternalServerError, "Proxy route upstream is invalid", err)
	}
	if c.httpProxyTransport == nil {
		c.httpProxyTransport = newConnectHTTPProxyTransport()
	}

	outReq := req.Clone(req.Context())
	outReq.RequestURI = ""
	outReq.URL = req.URL
	outReq.Host = req.URL.Host
	reqCtx, cancel := context.WithCancel(req.Context())
	defer cancel()
	responseTimeout := compiled.responseTimeout
	if responseTimeout <= 0 {
		responseTimeout = defaultConnectProxyResponseTimeout
	}
	timer := time.AfterFunc(responseTimeout, cancel)
	outReq = outReq.WithContext(context.WithValue(reqCtx, connectProxyURLKey{}, backendURL))

	response, err := c.httpProxyTransport.RoundTrip(outReq)
	stopped := timer.Stop()
	if !stopped && err == nil {
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		err = errors.New("proxy upstream response header timeout")
	}
	if err != nil {
		c.app.Logger().Warnf("[connect-proxy] failed to forward HTTP target %s via %s: %s", target, backendURL.Host, err)
		return c.failConnect(ctx, http.StatusBadGateway, "Failed to forward proxy request", err)
	}
	defer response.Body.Close()

	writeProxyResponse(ctx, response)
	elapsed := time.Since(started)
	c.logAccess(ctx, sni, backendURL.Host, req.Method, req.RequestURI, req.Proto, response.StatusCode, elapsed, accessLogMeta{
		UpstreamStatus:         response.StatusCode,
		UpstreamResponseLength: response.ContentLength,
		UpstreamResponseTime:   elapsed,
	})
	return false, true, nil
}

func (c *core) proxyRateLimited(ctx *zoox.Context, ruleIdx int) bool {
	if c.rateLimits == nil {
		return false
	}
	var rulePolicy *ratelimit.Policy
	if ruleIdx >= 0 && ruleIdx < len(c.rateLimits.ByRule) {
		rulePolicy = c.rateLimits.ByRule[ruleIdx]
	}
	result := ratelimit.Check(ctx.Request, c.rateLimits.Global, rulePolicy, ruleIdx)
	if !result.Blocked {
		return false
	}
	ratelimit.WriteResponseHeaders(ctx.SetHeader, ctx.Request, c.rateLimits.Global, rulePolicy, ruleIdx, result)
	ctx.Status(http.StatusTooManyRequests)
	ctx.String(http.StatusTooManyRequests, "Too Many Requests")
	return true
}

func (c *core) failConnect(ctx *zoox.Context, status int, message string, err error) (bool, bool, error) {
	if err != nil {
		c.app.Logger().Warnf("[connect-proxy] %s: %s", message, err)
	}
	if ctx != nil && ctx.Request != nil {
		host := ctx.Hostname()
		if ctx.Request.TLS != nil && ctx.Request.TLS.ServerName != "" {
			host = strings.ToLower(ctx.Request.TLS.ServerName)
		}
		path := ctx.Request.RequestURI
		if path == "" {
			path = connectTarget(ctx.Request)
		}
		c.logAccess(ctx, host, "-", ctx.Request.Method, path, ctx.Request.Proto, status, 0, accessLogMeta{
			UpstreamStatus:         status,
			UpstreamResponseLength: -1,
		})
	}
	c.writeProxyErrorResponse(ctx, status, message)
	return false, true, nil
}

func connectTarget(req *http.Request) string {
	if req.URL != nil && req.URL.Host != "" {
		return req.URL.Host
	}
	return req.Host
}

func targetAllowed(patterns []*regexp.Regexp, target string) bool {
	if len(patterns) == 0 {
		return true
	}
	for _, pattern := range patterns {
		location := pattern.FindStringIndex(target)
		if location != nil && location[0] == 0 && location[1] == len(target) {
			return true
		}
	}
	return false
}

func buildUpstreamConnect(req *http.Request, target string) *http.Request {
	upstream := req.Clone(context.WithoutCancel(req.Context()))
	upstream.Method = http.MethodConnect
	upstream.RequestURI = ""
	upstream.URL = &url.URL{Host: target}
	upstream.Body = nil
	upstream.ContentLength = 0
	upstream.Header.Set("Host", target)
	return upstream
}

func writeConnectRequestWithDeadline(conn net.Conn, req *http.Request, timeout time.Duration) error {
	if timeout > 0 {
		_ = conn.SetWriteDeadline(time.Now().Add(timeout))
		defer conn.SetWriteDeadline(time.Time{})
	}
	return req.Write(conn)
}

type connectProxyURLKey struct{}

func newConnectHTTPProxyTransport() *http.Transport {
	dialer := &net.Dialer{Timeout: defaultConnectProxyConnectTimeout, KeepAlive: 30 * time.Second}
	return &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			proxyURL, _ := req.Context().Value(connectProxyURLKey{}).(*url.URL)
			if proxyURL == nil {
				return nil, errors.New("CONNECT proxy upstream is not configured")
			}
			return proxyURL, nil
		},
		DialContext:           dialer.DialContext,
		DisableCompression:    true,
		ForceAttemptHTTP2:     false,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: time.Second,
	}
}

func proxyRequestAuthority(req *http.Request) string {
	if req == nil || req.URL == nil {
		return ""
	}
	host := strings.ToLower(req.URL.Hostname())
	if host == "" {
		return ""
	}
	port := req.URL.Port()
	if port == "" {
		port = "80"
	}
	return net.JoinHostPort(host, port)
}

var connectProxyHopHeaders = map[string]bool{
	"Connection":        true,
	"Proxy-Connection":  true,
	"Keep-Alive":        true,
	"Te":                true,
	"Trailer":           true,
	"Transfer-Encoding": true,
	"Upgrade":           true,
}

func copyProxyResponseHeaders(dst, src http.Header) {
	for key, values := range src {
		if connectProxyHopHeaders[http.CanonicalHeaderKey(key)] {
			continue
		}
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func writeProxyResponse(ctx *zoox.Context, response *http.Response) {
	copyProxyResponseHeaders(ctx.Writer.Header(), response.Header)
	ctx.Writer.WriteHeader(response.StatusCode)
	if response.Body != nil {
		_, _ = io.Copy(ctx.Writer, response.Body)
	}
}

type idleTimeoutConn struct {
	net.Conn
	timeout time.Duration
}

type idleTimeoutReader struct {
	conn    net.Conn
	reader  io.Reader
	timeout time.Duration
}

func (r *idleTimeoutReader) Read(p []byte) (int, error) {
	if r.timeout > 0 {
		_ = r.conn.SetReadDeadline(time.Now().Add(r.timeout))
	}
	return r.reader.Read(p)
}

func (c *idleTimeoutConn) Read(p []byte) (int, error) {
	if c.timeout > 0 {
		_ = c.Conn.SetReadDeadline(time.Now().Add(c.timeout))
	}
	return c.Conn.Read(p)
}

func (c *idleTimeoutConn) Write(p []byte) (int, error) {
	if c.timeout > 0 {
		_ = c.Conn.SetWriteDeadline(time.Now().Add(c.timeout))
	}
	return c.Conn.Write(p)
}

func (c *idleTimeoutConn) CloseWrite() error {
	if closer, ok := c.Conn.(interface{ CloseWrite() error }); ok {
		return closer.CloseWrite()
	}
	return nil
}

func wrapIdleTimeout(conn net.Conn, timeout time.Duration) net.Conn {
	if timeout <= 0 {
		return conn
	}
	return &idleTimeoutConn{Conn: conn, timeout: timeout}
}

func wrapIdleTimeoutReader(conn net.Conn, reader io.Reader, timeout time.Duration) io.Reader {
	if timeout <= 0 {
		return reader
	}
	return &idleTimeoutReader{conn: conn, reader: reader, timeout: timeout}
}

func tunnelConnectPair(client net.Conn, clientReader io.Reader, upstream net.Conn, upstreamReader io.Reader) {
	done := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(client, upstreamReader)
		closeWrite(client)
		done <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(upstream, clientReader)
		closeWrite(upstream)
		done <- struct{}{}
	}()
	<-done
	<-done
}

func closeWrite(conn net.Conn) {
	if closer, ok := conn.(interface{ CloseWrite() error }); ok {
		_ = closer.CloseWrite()
	}
}

func secondsOrDefault(seconds int64, fallback time.Duration) time.Duration {
	if seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return fallback
}
