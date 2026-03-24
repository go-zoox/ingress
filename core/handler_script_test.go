package core

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/zoox"
	"github.com/go-zoox/zoox/defaults"
)

func TestExecuteJavaScriptHandlerScript_AliasesAndSetHeader(t *testing.T) {
	ctx, rec := createTestZooxContext(http.MethodGet, "/test")
	handlerCfg := &rule.Handler{
		Type:       handlerTypeScript,
		Engine:     scriptEngineJavaScript,
		StatusCode: 200,
		Body:       "init",
		Headers:    map[string]string{},
		Script: `
if (ctx.method !== "GET") throw new Error("method alias failed")
if (ctx.path !== "/test") throw new Error("path alias failed")
if (ctx.headers["X-Test"] !== "1") throw new Error("headers alias failed")
ctx.status = 201
ctx.type = "application/json"
ctx.body = JSON.stringify({ ok: true })
ctx.setHeader("X-From-Ctx", "1")
ctx.response.setHeader("X-From-Response", "1")
`,
	}
	ctx.Request.Header.Set("X-Test", "1")

	if err := executeJavaScriptHandlerScript(ctx, handlerCfg); err != nil {
		t.Fatalf("executeJavaScriptHandlerScript failed: %v", err)
	}

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("expected content type application/json, got %s", rec.Header().Get("Content-Type"))
	}
	if rec.Body.String() != `{"ok":true}` {
		t.Fatalf("unexpected body: %q", rec.Body.String())
	}
	if rec.Header().Get("X-From-Ctx") != "1" {
		t.Fatalf("expected X-From-Ctx header")
	}
	if rec.Header().Get("X-From-Response") != "1" {
		t.Fatalf("expected X-From-Response header")
	}
}

func TestExecuteGoYaegiHandlerScript_UseZooxContext(t *testing.T) {
	ctx, _ := createTestZooxContext(http.MethodPost, "/go")
	script := `
ctx.Method = "PATCH"
ctx.Set("X-Engine", "go")
`

	if err := executeGoYaegiHandlerScript(ctx, script); err != nil {
		t.Fatalf("executeGoYaegiHandlerScript failed: %v", err)
	}

	if ctx.Method != "PATCH" {
		t.Fatalf("expected method patched, got %s", ctx.Method)
	}
	if ctx.Writer.Header().Get("X-Engine") != "go" {
		t.Fatalf("expected X-Engine header")
	}
}

func TestExecuteHandlerScript_UnsupportedEngine(t *testing.T) {
	ctx, _ := createTestZooxContext(http.MethodGet, "/")
	err := executeHandlerScript(ctx, &rule.Handler{
		Type:   handlerTypeScript,
		Engine: "unknown",
		Script: "",
	})
	if err == nil {
		t.Fatalf("expected unsupported engine error")
	}
	if !strings.Contains(err.Error(), "unsupported script engine") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecuteJavaScriptHandlerScript_Fetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("fetched-by-js"))
	}))
	defer server.Close()

	ctx, rec := createTestZooxContext(http.MethodGet, "/")
	handlerCfg := &rule.Handler{
		Type:   handlerTypeScript,
		Engine: scriptEngineJavaScript,
		Script: `ctx.body = ctx.fetch("` + server.URL + `")`,
	}

	if err := executeJavaScriptHandlerScript(ctx, handlerCfg); err != nil {
		t.Fatalf("executeJavaScriptHandlerScript failed: %v", err)
	}
	if rec.Body.String() != "fetched-by-js" {
		t.Fatalf("unexpected body: %q", rec.Body.String())
	}
}

func TestExecuteGoYaegiHandlerScript_Fetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("fetched-by-go"))
	}))
	defer server.Close()

	ctx, rec := createTestZooxContext(http.MethodGet, "/")
	script := `
res, err := ctx.Fetch().Get("` + server.URL + `", nil).Execute()
if err != nil {
	panic(err)
}
ctx.String(200, res.String())
`
	if err := executeGoYaegiHandlerScript(ctx, script); err != nil {
		t.Fatalf("executeGoYaegiHandlerScript failed: %v", err)
	}
	if rec.Body.String() != "fetched-by-go" {
		t.Fatalf("unexpected body: %q", rec.Body.String())
	}
}

func createTestZooxContext(method, path string) (*zoox.Context, *httptest.ResponseRecorder) {
	app := defaults.Default()
	req := httptest.NewRequest(method, "http://localhost"+path, nil)
	rec := httptest.NewRecorder()

	var captured *zoox.Context
	app.Use(func(ctx *zoox.Context) {
		captured = ctx
	})
	app.ServeHTTP(rec, req)
	return captured, rec
}
