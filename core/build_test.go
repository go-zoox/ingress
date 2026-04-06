package core

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/ingress/core/service"
)

func TestBuild_AccessLogExtraFields_WithTLS(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://example.com/orders?id=1", nil)
	req.Header.Set("Referer", "https://portal.example.com/list")
	req.Header.Set("User-Agent", "ingress-test-agent")
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
	req.TLS = &tls.ConnectionState{
		Version:      tls.VersionTLS13,
		CipherSuite:  tls.TLS_AES_128_GCM_SHA256,
		NegotiatedProtocol: "h2",
	}

	extra := buildAccessLogExtraFields(req, 200, 456, 123*time.Millisecond)

	required := []string{
		`referer="https://portal.example.com/list"`,
		`ua="ingress-test-agent"`,
		`xff="10.0.0.1, 10.0.0.2"`,
		`tls_protocol="TLS 1.3"`,
		`tls_cipher="TLS_AES_128_GCM_SHA256"`,
		`upstream_status=200`,
		`upstream_response_length=456`,
		`upstream_response_time=123ms`,
	}

	for _, item := range required {
		if !strings.Contains(extra, item) {
			t.Fatalf("expected extra fields to contain %q, got: %s", item, extra)
		}
	}
}

func TestBuild_AccessLogExtraFields_WithoutTLS(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/health", nil)

	extra := buildAccessLogExtraFields(req, 503, -1, 27*time.Millisecond)

	required := []string{
		`referer="-"`,
		`ua="-"`,
		`xff="-"`,
		`tls_protocol="-"`,
		`tls_cipher="-"`,
		`upstream_status=503`,
		`upstream_response_length=-1`,
		`upstream_response_time=27ms`,
	}

	for _, item := range required {
		if !strings.Contains(extra, item) {
			t.Fatalf("expected extra fields to contain %q, got: %s", item, extra)
		}
	}
}

func TestBuild_RequestDelay(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "test.example.com",
				Backend: rule.Backend{
					Service: service.Service{
						Name:     "test-service",
						Port:     80,
						Protocol: "http",
						Request: service.Request{
							Delay: 100, // 100ms delay
						},
					},
				},
			},
		},
	}

	c, err := New("test-version", cfg)
	if err != nil {
		t.Fatalf("failed to create core: %v", err)
	}

	if c == nil {
		t.Fatal("expected core instance, got nil")
	}

	// Verify the service has delay configured
	matchedService, err := MatchHost(cfg.Rules, rule.Backend{}, "test.example.com")
	if err != nil {
		t.Fatalf("failed to match host: %v", err)
	}

	if matchedService.Service.Request.Delay != 100 {
		t.Errorf("expected delay 100ms, got %d", matchedService.Service.Request.Delay)
	}

	// Verify delay duration conversion
	delayDuration := time.Duration(matchedService.Service.Request.Delay) * time.Millisecond
	expectedDuration := 100 * time.Millisecond
	if delayDuration != expectedDuration {
		t.Errorf("expected delay duration %v, got %v", expectedDuration, delayDuration)
	}
}

func TestBuild_RequestTimeout(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "test.example.com",
				Backend: rule.Backend{
					Service: service.Service{
						Name:     "test-service",
						Port:     80,
						Protocol: "http",
						Request: service.Request{
							Timeout: 30, // 30 seconds timeout
						},
					},
				},
			},
		},
	}

	c, err := New("test-version", cfg)
	if err != nil {
		t.Fatalf("failed to create core: %v", err)
	}

	if c == nil {
		t.Fatal("expected core instance, got nil")
	}

	// Verify the service has timeout configured
	matchedService, err := MatchHost(cfg.Rules, rule.Backend{}, "test.example.com")
	if err != nil {
		t.Fatalf("failed to match host: %v", err)
	}

	if matchedService.Service.Request.Timeout != 30 {
		t.Errorf("expected timeout 30s, got %d", matchedService.Service.Request.Timeout)
	}

	// Verify timeout duration conversion
	timeoutDuration := time.Duration(matchedService.Service.Request.Timeout) * time.Second
	expectedDuration := 30 * time.Second
	if timeoutDuration != expectedDuration {
		t.Errorf("expected timeout duration %v, got %v", expectedDuration, timeoutDuration)
	}
}

func TestBuild_RequestDelayAndTimeout(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "test.example.com",
				Backend: rule.Backend{
					Service: service.Service{
						Name:     "test-service",
						Port:     80,
						Protocol: "http",
						Request: service.Request{
							Delay:   200, // 200ms delay
							Timeout: 60,  // 60 seconds timeout
						},
					},
				},
			},
		},
	}

	c, err := New("test-version", cfg)
	if err != nil {
		t.Fatalf("failed to create core: %v", err)
	}

	if c == nil {
		t.Fatal("expected core instance, got nil")
	}

	// Verify the service has both delay and timeout configured
	matchedService, err := MatchHost(cfg.Rules, rule.Backend{}, "test.example.com")
	if err != nil {
		t.Fatalf("failed to match host: %v", err)
	}

	if matchedService.Service.Request.Delay != 200 {
		t.Errorf("expected delay 200ms, got %d", matchedService.Service.Request.Delay)
	}

	if matchedService.Service.Request.Timeout != 60 {
		t.Errorf("expected timeout 60s, got %d", matchedService.Service.Request.Timeout)
	}

	// Verify both duration conversions
	delayDuration := time.Duration(matchedService.Service.Request.Delay) * time.Millisecond
	timeoutDuration := time.Duration(matchedService.Service.Request.Timeout) * time.Second

	if delayDuration != 200*time.Millisecond {
		t.Errorf("expected delay duration 200ms, got %v", delayDuration)
	}

	if timeoutDuration != 60*time.Second {
		t.Errorf("expected timeout duration 60s, got %v", timeoutDuration)
	}
}

func TestBuild_RequestTimeoutContext(t *testing.T) {
	// Test that timeout creates a context with timeout
	timeout := int64(5) // 5 seconds
	timeoutDuration := time.Duration(timeout) * time.Second

	// Create a base context
	baseCtx := context.Background()

	// Create timeout context (simulating what build.go does)
	timeoutCtx, cancel := context.WithTimeout(baseCtx, timeoutDuration)
	defer cancel()

	// Verify the context has a deadline
	deadline, ok := timeoutCtx.Deadline()
	if !ok {
		t.Fatal("expected context to have a deadline")
	}

	// Verify the deadline is approximately timeoutDuration from now
	expectedDeadline := time.Now().Add(timeoutDuration)
	diff := expectedDeadline.Sub(deadline)
	if diff < -time.Second || diff > time.Second {
		t.Errorf("expected deadline to be approximately %v from now, got %v (diff: %v)", timeoutDuration, deadline, diff)
	}

	// Verify context is not done initially
	select {
	case <-timeoutCtx.Done():
		t.Fatal("expected context to not be done initially")
	default:
		// Good, context is not done
	}
}

func TestBuild_RequestDelayTiming(t *testing.T) {
	// Test that delay actually causes a delay
	delay := int64(100) // 100ms
	delayDuration := time.Duration(delay) * time.Millisecond

	start := time.Now()

	// Simulate delay (what build.go does)
	time.Sleep(delayDuration)

	elapsed := time.Since(start)

	// Verify the delay was approximately correct (allow some margin for timing)
	if elapsed < delayDuration-time.Millisecond*10 {
		t.Errorf("expected delay of at least %v, got %v", delayDuration-time.Millisecond*10, elapsed)
	}

	if elapsed > delayDuration+time.Millisecond*50 {
		t.Errorf("expected delay of at most %v, got %v", delayDuration+time.Millisecond*50, elapsed)
	}
}

func TestBuild_RequestTimeoutExpiration(t *testing.T) {
	// Test that timeout context expires correctly
	timeout := int64(1) // 1 second
	timeoutDuration := time.Duration(timeout) * time.Second

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	// Verify context expires after timeout
	select {
	case <-timeoutCtx.Done():
		// Good, context expired
		elapsed := time.Since(time.Now().Add(-timeoutDuration))
		if elapsed < 0 {
			elapsed = -elapsed
		}
		if elapsed > time.Second*2 {
			t.Errorf("context expired too late, elapsed: %v", elapsed)
		}
	case <-time.After(timeoutDuration + time.Second):
		t.Fatal("expected context to expire within timeout duration")
	}
}

func TestBuild_RequestWithZeroDelayAndTimeout(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "test.example.com",
				Backend: rule.Backend{
					Service: service.Service{
						Name:     "test-service",
						Port:     80,
						Protocol: "http",
						Request: service.Request{
							Delay:   0, // No delay
							Timeout: 0, // No timeout
						},
					},
				},
			},
		},
	}

	c, err := New("test-version", cfg)
	if err != nil {
		t.Fatalf("failed to create core: %v", err)
	}

	if c == nil {
		t.Fatal("expected core instance, got nil")
	}

	// Verify zero values are handled correctly
	matchedService, err := MatchHost(cfg.Rules, rule.Backend{}, "test.example.com")
	if err != nil {
		t.Fatalf("failed to match host: %v", err)
	}

	if matchedService.Service.Request.Delay != 0 {
		t.Errorf("expected delay 0, got %d", matchedService.Service.Request.Delay)
	}

	if matchedService.Service.Request.Timeout != 0 {
		t.Errorf("expected timeout 0, got %d", matchedService.Service.Request.Timeout)
	}
}

func TestBuild_RequestTimeoutInHTTPRequest(t *testing.T) {
	// Test that timeout is applied to HTTP request context
	timeout := int64(5) // 5 seconds
	timeoutDuration := time.Duration(timeout) * time.Second

	// Create a base HTTP request
	req, err := http.NewRequest("GET", "http://example.com", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	// Apply timeout to request context (simulating what build.go does in OnRequest)
	timeoutCtx, cancel := context.WithTimeout(req.Context(), timeoutDuration)
	_ = cancel // cancel will be called when request completes
	req = req.WithContext(timeoutCtx)

	// Verify the request context has a deadline
	deadline, ok := req.Context().Deadline()
	if !ok {
		t.Fatal("expected request context to have a deadline")
	}

	// Verify the deadline is approximately timeoutDuration from now
	expectedDeadline := time.Now().Add(timeoutDuration)
	diff := expectedDeadline.Sub(deadline)
	if diff < -time.Second || diff > time.Second {
		t.Errorf("expected deadline to be approximately %v from now, got %v (diff: %v)", timeoutDuration, deadline, diff)
	}
}

func TestBuild_HandlerBackendStatusCode(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "handler.example.work",
				Backend: rule.Backend{
					Type: backendTypeHandler,
					Handler: rule.Handler{
						StatusCode: 201,
						Body:       "created",
					},
				},
			},
		},
	}

	c, err := New("test-version", cfg)
	if err != nil {
		t.Fatalf("failed to create core: %v", err)
	}

	ins, ok := c.(*core)
	if !ok {
		t.Fatalf("failed to cast core instance")
	}
	if err := ins.build(); err != nil {
		t.Fatalf("failed to build core: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://handler.example.work/", nil)
	recorder := httptest.NewRecorder()
	ins.app.ServeHTTP(recorder, req)

	if recorder.Code != 201 {
		t.Fatalf("expected status code 201, got %d", recorder.Code)
	}
	if body := recorder.Body.String(); body != "created" {
		t.Fatalf("expected body 'created', got %q", body)
	}
}

func TestBuild_HandlerBackendHeadersAndJSONBody(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "handler.example.work",
				Backend: rule.Backend{
					Service: service.Service{
						Name: "upstream-service",
						Port: 8080,
					},
				},
				Paths: []rule.Path{
					{
						Path: "/custom/handler/json",
						Backend: rule.Backend{
							Type: backendTypeHandler,
							Handler: rule.Handler{
								Headers: map[string]string{
									"Content-Type": "application/json",
								},
								Body: `{"message":"Hello, World!"}`,
							},
						},
					},
				},
			},
		},
	}

	c, err := New("test-version", cfg)
	if err != nil {
		t.Fatalf("failed to create core: %v", err)
	}

	ins, ok := c.(*core)
	if !ok {
		t.Fatalf("failed to cast core instance")
	}
	if err := ins.build(); err != nil {
		t.Fatalf("failed to build core: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://handler.example.work/custom/handler/json", nil)
	recorder := httptest.NewRecorder()
	ins.app.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status code 200, got %d", recorder.Code)
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.Contains(contentType, "application/json") {
		t.Fatalf("expected Content-Type contains application/json, got %q", contentType)
	}
	if body := recorder.Body.String(); body != `{"message":"Hello, World!"}` {
		t.Fatalf("expected json body, got %q", body)
	}
}

func TestBuild_HandlerBackendFileServer(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, "home.html"), []byte("file-server-home"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "asset.txt"), []byte("asset-content"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "handler.example.work",
				Backend: rule.Backend{
					Type: backendTypeHandler,
					Handler: rule.Handler{
						Type:      handlerTypeFileServer,
						RootDir:   tempDir,
						IndexFile: "home.html",
					},
				},
			},
		},
	}

	c, err := New("test-version", cfg)
	if err != nil {
		t.Fatalf("failed to create core: %v", err)
	}
	ins := c.(*core)
	if err := ins.build(); err != nil {
		t.Fatalf("failed to build core: %v", err)
	}

	reqIndex := httptest.NewRequest(http.MethodGet, "http://handler.example.work/", nil)
	recIndex := httptest.NewRecorder()
	ins.app.ServeHTTP(recIndex, reqIndex)
	if recIndex.Code != http.StatusOK {
		t.Fatalf("expected status code 200, got %d", recIndex.Code)
	}
	if body := recIndex.Body.String(); body != "file-server-home" {
		t.Fatalf("expected index content, got %q", body)
	}

	reqFile := httptest.NewRequest(http.MethodGet, "http://handler.example.work/asset.txt", nil)
	recFile := httptest.NewRecorder()
	ins.app.ServeHTTP(recFile, reqFile)
	if recFile.Code != http.StatusOK {
		t.Fatalf("expected status code 200, got %d", recFile.Code)
	}
	if body := recFile.Body.String(); body != "asset-content" {
		t.Fatalf("expected file content, got %q", body)
	}
}

func TestBuild_HandlerBackendTemplates(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, "index.html"), []byte("<h1>{{.Path}}</h1>"), 0o644); err != nil {
		t.Fatalf("failed to write template: %v", err)
	}

	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "handler.example.work",
				Backend: rule.Backend{
					Type: backendTypeHandler,
					Handler: rule.Handler{
						Type:    handlerTypeTemplates,
						RootDir: tempDir,
					},
				},
			},
		},
	}

	c, err := New("test-version", cfg)
	if err != nil {
		t.Fatalf("failed to create core: %v", err)
	}
	ins := c.(*core)
	if err := ins.build(); err != nil {
		t.Fatalf("failed to build core: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://handler.example.work/", nil)
	rec := httptest.NewRecorder()
	ins.app.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status code 200, got %d", rec.Code)
	}
	if body := rec.Body.String(); body != "<h1>/</h1>" {
		t.Fatalf("expected rendered template body, got %q", body)
	}
}

func TestBuild_HandlerBackendScriptJavaScript(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "handler.example.work",
				Backend: rule.Backend{
					Type: backendTypeHandler,
					Handler: rule.Handler{
						Type:   handlerTypeScript,
						Engine: scriptEngineJavaScript,
						Script: `
ctx.status = 202
ctx.type = "application/json"
ctx.body = JSON.stringify({
  method: ctx.method,
  path: ctx.path,
})
ctx.setHeader("X-Script", "javascript")
ctx.response.setHeader("X-Response", "ok")
`,
					},
				},
			},
		},
	}

	c, err := New("test-version", cfg)
	if err != nil {
		t.Fatalf("failed to create core: %v", err)
	}
	ins := c.(*core)
	if err := ins.build(); err != nil {
		t.Fatalf("failed to build core: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://handler.example.work/demo", nil)
	rec := httptest.NewRecorder()
	ins.app.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status code 202, got %d", rec.Code)
	}
	if contentType := rec.Header().Get("Content-Type"); !strings.Contains(contentType, "application/json") {
		t.Fatalf("expected Content-Type contains application/json, got %q", contentType)
	}
	if rec.Header().Get("X-Script") != "javascript" {
		t.Fatalf("expected X-Script header")
	}
	if rec.Header().Get("X-Response") != "ok" {
		t.Fatalf("expected X-Response header")
	}
	if body := rec.Body.String(); body != `{"method":"GET","path":"/demo"}` {
		t.Fatalf("unexpected response body: %q", body)
	}
}

func TestBuild_HandlerBackendScriptGo(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "handler.example.work",
				Backend: rule.Backend{
					Type: backendTypeHandler,
					Handler: rule.Handler{
						Type:   handlerTypeScript,
						Engine: scriptEngineGo,
						Script: `
ctx.SetHeader("X-Handler-Engine", "go")
ctx.String(200, ctx.Method+" "+ctx.Path)
`,
					},
				},
			},
		},
	}

	c, err := New("test-version", cfg)
	if err != nil {
		t.Fatalf("failed to create core: %v", err)
	}
	ins := c.(*core)
	if err := ins.build(); err != nil {
		t.Fatalf("failed to build core: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "http://handler.example.work/go-script", nil)
	rec := httptest.NewRecorder()
	ins.app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status code 200, got %d", rec.Code)
	}
	if rec.Header().Get("X-Handler-Engine") != "go" {
		t.Fatalf("expected X-Handler-Engine header")
	}
	if body := rec.Body.String(); body != "POST /go-script" {
		t.Fatalf("unexpected response body: %q", body)
	}
}

func TestBuild_HandlerBackendScriptJavaScriptStatusAlias(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "handler.example.work",
				Backend: rule.Backend{
					Type: backendTypeHandler,
					Handler: rule.Handler{
						Type:   handlerTypeScript,
						Engine: scriptEngineJavaScript,
						Script: `
ctx.response.status_code = 204
if (ctx.status !== 204) {
  throw new Error("ctx.status alias getter failed")
}
ctx.status = 206
ctx.type = "text/plain"
ctx.body = "status=" + ctx.status
`,
					},
				},
			},
		},
	}

	c, err := New("test-version", cfg)
	if err != nil {
		t.Fatalf("failed to create core: %v", err)
	}
	ins := c.(*core)
	if err := ins.build(); err != nil {
		t.Fatalf("failed to build core: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://handler.example.work/status-alias", nil)
	rec := httptest.NewRecorder()
	ins.app.ServeHTTP(rec, req)

	if rec.Code != http.StatusPartialContent {
		t.Fatalf("expected status code 206, got %d", rec.Code)
	}
	if body := rec.Body.String(); body != "status=206" {
		t.Fatalf("unexpected response body: %q", body)
	}
}

func TestBuild_HandlerBackendScriptJavaScriptResponseSetHeader(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "handler.example.work",
				Backend: rule.Backend{
					Type: backendTypeHandler,
					Handler: rule.Handler{
						Type:   handlerTypeScript,
						Engine: scriptEngineJavaScript,
						Script: `
ctx.response.setHeader("X-From-Response", "1")
ctx.setHeader("X-From-Ctx", "1")
ctx.body = "ok"
`,
					},
				},
			},
		},
	}

	c, err := New("test-version", cfg)
	if err != nil {
		t.Fatalf("failed to create core: %v", err)
	}
	ins := c.(*core)
	if err := ins.build(); err != nil {
		t.Fatalf("failed to build core: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://handler.example.work/headers", nil)
	rec := httptest.NewRecorder()
	ins.app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status code 200, got %d", rec.Code)
	}
	if rec.Header().Get("X-From-Response") != "1" {
		t.Fatalf("expected X-From-Response header")
	}
	if rec.Header().Get("X-From-Ctx") != "1" {
		t.Fatalf("expected X-From-Ctx header")
	}
}

func TestBuild_HTTP2HTTP3ZooxConfig(t *testing.T) {
	cfg := &Config{
		Port:      8080,
		EnableH2C: true,
		HTTPS: HTTPS{
			Port:              8443,
			EnableHTTP3:       true,
			HTTP3Port:         8443,
			HTTP3AltSvcMaxAge: 3600,
		},
		Rules: []rule.Rule{
			{
				Host: "h.example.com",
				Backend: rule.Backend{
					Service: service.Service{
						Name:     "svc",
						Port:     80,
						Protocol: "http",
					},
				},
			},
		},
	}

	c, err := New("test-version", cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ins := c.(*core)
	if err := ins.build(); err != nil {
		t.Fatalf("build: %v", err)
	}

	if !ins.app.Config.EnableH2C {
		t.Error("expected EnableH2C propagated to zoox")
	}
	if !ins.app.Config.EnableHTTP3 {
		t.Error("expected EnableHTTP3 propagated to zoox")
	}
	if ins.app.Config.HTTP3Port != 8443 {
		t.Errorf("HTTP3Port: got %d want 8443", ins.app.Config.HTTP3Port)
	}
	if ins.app.Config.HTTP3AltSvcMaxAge != 3600 {
		t.Errorf("HTTP3AltSvcMaxAge: got %d want 3600", ins.app.Config.HTTP3AltSvcMaxAge)
	}
}
