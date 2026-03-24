package core

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/dop251/goja"
	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/zoox"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

type scriptRequestContext struct {
	Method  string
	Path    string
	Headers map[string]string
}

type scriptResponseContext struct {
	StatusCode  int64
	ContentType string
	Headers     map[string]string
	Body        string
}

type scriptContext struct {
	Request  scriptRequestContext
	Response scriptResponseContext
}

func executeHandlerScript(ctx *zoox.Context, handlerCfg *rule.Handler) (*scriptResponseContext, error) {
	engine := handlerCfg.Engine
	if engine == "" {
		engine = scriptEngineJavaScript
	}

	requestHeaders := map[string]string{}
	for key, values := range ctx.Request.Header {
		requestHeaders[key] = strings.Join(values, ",")
	}

	responseHeaders := map[string]string{}
	for key, value := range handlerCfg.Headers {
		responseHeaders[key] = value
	}

	statusCode := handlerCfg.StatusCode
	if statusCode == 0 {
		statusCode = 200
	}
	contentType := responseHeaders["Content-Type"]

	runtimeCtx := &scriptContext{
		Request: scriptRequestContext{
			Method:  ctx.Method,
			Path:    ctx.Path,
			Headers: requestHeaders,
		},
		Response: scriptResponseContext{
			StatusCode:  statusCode,
			ContentType: contentType,
			Headers:     responseHeaders,
			Body:        handlerCfg.Body,
		},
	}

	switch engine {
	case scriptEngineJavaScript:
		if err := executeJavaScriptHandlerScript(runtimeCtx, handlerCfg.Script); err != nil {
			return nil, err
		}
	case scriptEngineGo:
		if err := executeGoYaegiHandlerScript(runtimeCtx, handlerCfg.Script); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported script engine: %s", engine)
	}

	if runtimeCtx.Response.ContentType != "" {
		runtimeCtx.Response.Headers["Content-Type"] = runtimeCtx.Response.ContentType
	}

	return &runtimeCtx.Response, nil
}

func executeGoYaegiHandlerScript(runtimeCtx *scriptContext, script string) error {
	i := interp.New(interp.Options{})
	i.Use(stdlib.Symbols)
	scriptWithPrelude := fmt.Sprintf(`
type Request struct {
	Method string
	Path string
	Headers map[string]string
}
type Response struct {
	StatusCode int64
	ContentType string
	Headers map[string]string
	Body string
}
type Context struct {
	Request Request
	Response Response
}
var ctx = &Context{
	Request: Request{
		Method: %s,
		Path: %s,
		Headers: %s,
	},
	Response: Response{
		StatusCode: %d,
		ContentType: %s,
		Headers: %s,
		Body: %s,
	},
}
func __run(){
%s
}
`, strconv.Quote(runtimeCtx.Request.Method), strconv.Quote(runtimeCtx.Request.Path), toGoStringMapLiteral(runtimeCtx.Request.Headers), runtimeCtx.Response.StatusCode, strconv.Quote(runtimeCtx.Response.ContentType), toGoStringMapLiteral(runtimeCtx.Response.Headers), strconv.Quote(runtimeCtx.Response.Body), script)
	if _, err := i.Eval(scriptWithPrelude); err != nil {
		return err
	}
	_, err := i.Eval("__run()")
	if err != nil {
		return err
	}

	ctxValue, err := i.Eval("ctx")
	if err != nil {
		return err
	}

	ctxReflectValue := reflect.Indirect(ctxValue)
	responseValue := ctxReflectValue.FieldByName("Response")
	runtimeCtx.Response.StatusCode = responseValue.FieldByName("StatusCode").Int()
	runtimeCtx.Response.ContentType = responseValue.FieldByName("ContentType").String()
	runtimeCtx.Response.Body = responseValue.FieldByName("Body").String()

	headersValue := responseValue.FieldByName("Headers")
	runtimeCtx.Response.Headers = map[string]string{}
	for _, key := range headersValue.MapKeys() {
		runtimeCtx.Response.Headers[key.String()] = headersValue.MapIndex(key).String()
	}

	return nil
}

func toGoStringMapLiteral(input map[string]string) string {
	if len(input) == 0 {
		return "map[string]string{}"
	}

	parts := make([]string, 0, len(input))
	for key, value := range input {
		parts = append(parts, fmt.Sprintf("%s: %s", strconv.Quote(key), strconv.Quote(value)))
	}
	return fmt.Sprintf("map[string]string{%s}", strings.Join(parts, ", "))
}

func executeJavaScriptHandlerScript(runtimeCtx *scriptContext, script string) error {
	vm := goja.New()
	ctxObject := vm.NewObject()
	requestObject := vm.NewObject()
	responseObject := vm.NewObject()
	requestHeadersObject := vm.NewObject()
	responseHeadersObject := vm.NewObject()

	for key, value := range runtimeCtx.Request.Headers {
		if err := requestHeadersObject.Set(key, value); err != nil {
			return err
		}
	}
	for key, value := range runtimeCtx.Response.Headers {
		if err := responseHeadersObject.Set(key, value); err != nil {
			return err
		}
	}

	if err := requestObject.Set("method", runtimeCtx.Request.Method); err != nil {
		return err
	}
	if err := requestObject.Set("path", runtimeCtx.Request.Path); err != nil {
		return err
	}
	if err := requestObject.Set("headers", requestHeadersObject); err != nil {
		return err
	}

	if err := responseObject.Set("status_code", runtimeCtx.Response.StatusCode); err != nil {
		return err
	}
	if err := responseObject.Set("content_type", runtimeCtx.Response.ContentType); err != nil {
		return err
	}
	if err := responseObject.Set("headers", responseHeadersObject); err != nil {
		return err
	}
	if err := responseObject.Set("body", runtimeCtx.Response.Body); err != nil {
		return err
	}

	setHeader := func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) >= 2 {
			key := call.Argument(0).String()
			value := call.Argument(1).String()
			_ = responseHeadersObject.Set(key, value)
			if strings.EqualFold(key, "Content-Type") {
				_ = responseObject.Set("content_type", value)
			}
		}
		return goja.Undefined()
	}

	if err := responseObject.Set("setHeader", setHeader); err != nil {
		return err
	}
	if err := ctxObject.Set("setHeader", setHeader); err != nil {
		return err
	}

	if err := ctxObject.Set("request", requestObject); err != nil {
		return err
	}
	if err := ctxObject.Set("response", responseObject); err != nil {
		return err
	}
	if err := vm.Set("ctx", ctxObject); err != nil {
		return err
	}

	if _, err := vm.RunString(`
Object.defineProperty(ctx, "method", { get: function() { return ctx.request.method; }});
Object.defineProperty(ctx, "path", { get: function() { return ctx.request.path; }});
Object.defineProperty(ctx, "headers", { get: function() { return ctx.request.headers; }});
Object.defineProperty(ctx, "type", {
	get: function() { return ctx.response.content_type; },
	set: function(v) { ctx.response.content_type = v; }
});
Object.defineProperty(ctx, "status", {
	get: function() { return ctx.response.status_code; },
	set: function(v) { ctx.response.status_code = v; }
});
Object.defineProperty(ctx, "body", {
	get: function() { return ctx.response.body; },
	set: function(v) { ctx.response.body = v; }
});
`); err != nil {
		return err
	}

	if _, err := vm.RunString(script); err != nil {
		return err
	}

	runtimeCtx.Response.StatusCode = responseObject.Get("status_code").ToInteger()
	runtimeCtx.Response.ContentType = responseObject.Get("content_type").String()
	runtimeCtx.Response.Body = responseObject.Get("body").String()
	runtimeCtx.Response.Headers = map[string]string{}
	for _, key := range responseHeadersObject.Keys() {
		runtimeCtx.Response.Headers[key] = responseHeadersObject.Get(key).String()
	}

	return nil
}
