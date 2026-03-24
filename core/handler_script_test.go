package core

import (
	"strings"
	"testing"
)

func TestExecuteJavaScriptHandlerScript_AliasesAndSetHeader(t *testing.T) {
	runtimeCtx := &scriptContext{
		Request: scriptRequestContext{
			Method: "GET",
			Path:   "/test",
			Headers: map[string]string{
				"X-Test": "1",
			},
		},
		Response: scriptResponseContext{
			StatusCode:  200,
			ContentType: "text/plain",
			Headers:     map[string]string{},
			Body:        "init",
		},
	}

	script := `
if (ctx.method !== "GET") throw new Error("method alias failed")
if (ctx.path !== "/test") throw new Error("path alias failed")
if (ctx.headers["X-Test"] !== "1") throw new Error("headers alias failed")
ctx.status = 201
ctx.type = "application/json"
ctx.body = JSON.stringify({ ok: true })
ctx.setHeader("X-From-Ctx", "1")
ctx.response.setHeader("X-From-Response", "1")
`

	if err := executeJavaScriptHandlerScript(runtimeCtx, script); err != nil {
		t.Fatalf("executeJavaScriptHandlerScript failed: %v", err)
	}

	if runtimeCtx.Response.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", runtimeCtx.Response.StatusCode)
	}
	if runtimeCtx.Response.ContentType != "application/json" {
		t.Fatalf("expected content type application/json, got %s", runtimeCtx.Response.ContentType)
	}
	if runtimeCtx.Response.Body != `{"ok":true}` {
		t.Fatalf("unexpected body: %q", runtimeCtx.Response.Body)
	}
	if runtimeCtx.Response.Headers["X-From-Ctx"] != "1" {
		t.Fatalf("expected X-From-Ctx header")
	}
	if runtimeCtx.Response.Headers["X-From-Response"] != "1" {
		t.Fatalf("expected X-From-Response header")
	}
}

func TestExecuteGoYaegiHandlerScript_MutateResponse(t *testing.T) {
	runtimeCtx := &scriptContext{
		Request: scriptRequestContext{
			Method: "POST",
			Path:   "/go",
			Headers: map[string]string{
				"X-Test": "1",
			},
		},
		Response: scriptResponseContext{
			StatusCode:  200,
			ContentType: "text/plain",
			Headers:     map[string]string{},
			Body:        "",
		},
	}

	script := `
ctx.Response.StatusCode = 202
ctx.Response.ContentType = "application/json"
ctx.Response.Headers["X-Engine"] = "go"
ctx.Response.Body = ctx.Request.Method + " " + ctx.Request.Path
`

	if err := executeGoYaegiHandlerScript(runtimeCtx, script); err != nil {
		t.Fatalf("executeGoYaegiHandlerScript failed: %v", err)
	}

	if runtimeCtx.Response.StatusCode != 202 {
		t.Fatalf("expected status 202, got %d", runtimeCtx.Response.StatusCode)
	}
	if runtimeCtx.Response.ContentType != "application/json" {
		t.Fatalf("expected content type application/json, got %s", runtimeCtx.Response.ContentType)
	}
	if runtimeCtx.Response.Headers["X-Engine"] != "go" {
		t.Fatalf("expected X-Engine header")
	}
	if runtimeCtx.Response.Body != "POST /go" {
		t.Fatalf("unexpected body: %q", runtimeCtx.Response.Body)
	}
}

func TestExecuteHandlerScript_UnsupportedEngine(t *testing.T) {
	runtimeCtx := &scriptContext{
		Request: scriptRequestContext{Method: "GET", Path: "/"},
		Response: scriptResponseContext{
			StatusCode: 200,
			Headers:    map[string]string{},
		},
	}

	_, err := executeHandlerScriptWithContext(runtimeCtx, "unknown", "")
	if err == nil {
		t.Fatalf("expected unsupported engine error")
	}
	if !strings.Contains(err.Error(), "unsupported script engine") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestToGoStringMapLiteral(t *testing.T) {
	if got := toGoStringMapLiteral(nil); got != "map[string]string{}" {
		t.Fatalf("unexpected nil map literal: %s", got)
	}

	got := toGoStringMapLiteral(map[string]string{
		`a"b`: `c\d`,
	})
	if !strings.HasPrefix(got, "map[string]string{") || !strings.HasSuffix(got, "}") {
		t.Fatalf("unexpected map literal format: %s", got)
	}
	if !strings.Contains(got, `"a\"b"`) {
		t.Fatalf("expected escaped key in literal: %s", got)
	}
	if !strings.Contains(got, `"c\\d"`) {
		t.Fatalf("expected escaped value in literal: %s", got)
	}
}

func executeHandlerScriptWithContext(runtimeCtx *scriptContext, engine string, script string) (*scriptResponseContext, error) {
	handlerCfg := &struct {
		Engine string
		Script string
	}{
		Engine: engine,
		Script: script,
	}

	switch handlerCfg.Engine {
	case scriptEngineJavaScript:
		if err := executeJavaScriptHandlerScript(runtimeCtx, handlerCfg.Script); err != nil {
			return nil, err
		}
	case scriptEngineGo:
		if err := executeGoYaegiHandlerScript(runtimeCtx, handlerCfg.Script); err != nil {
			return nil, err
		}
	default:
		return nil, &unsupportedEngineError{engine: handlerCfg.Engine}
	}

	return &runtimeCtx.Response, nil
}

type unsupportedEngineError struct {
	engine string
}

func (e *unsupportedEngineError) Error() string {
	return "unsupported script engine: " + e.engine
}
