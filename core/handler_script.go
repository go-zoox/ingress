package core

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/dop251/goja"
	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/zoox"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

func executeHandlerScript(ctx *zoox.Context, handlerCfg *rule.Handler) error {
	engine := handlerCfg.Engine
	if engine == "" {
		engine = scriptEngineJavaScript
	}

	switch engine {
	case scriptEngineJavaScript:
		if err := executeJavaScriptHandlerScript(ctx, handlerCfg); err != nil {
			return err
		}
	case scriptEngineGo:
		if err := executeGoYaegiHandlerScript(ctx, handlerCfg.Script); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported script engine: %s", engine)
	}

	return nil
}

func executeGoYaegiHandlerScript(ctx *zoox.Context, script string) error {
	i := interp.New(interp.Options{
		GoPath: getGoPath(),
	})
	i.Use(stdlib.Symbols)
	i.Use(map[string]map[string]reflect.Value{
		"ingressctx/ingressctx": {
			"GetCtx": reflect.ValueOf(func() *zoox.Context {
				return ctx
			}),
		},
	})
	scriptWithPrelude := fmt.Sprintf(`import "ingressctx"
func __run() {
	ctx := ingressctx.GetCtx()
%s
}
`, script)
	if _, err := i.Eval(scriptWithPrelude); err != nil {
		return err
	}
	_, err := i.Eval("__run()")
	return err
}

func getGoPath() string {
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		return gopath
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, ".go")
}

func executeJavaScriptHandlerScript(ctx *zoox.Context, handlerCfg *rule.Handler) error {
	responseHeaders := map[string]string{}
	for key, value := range handlerCfg.Headers {
		responseHeaders[key] = value
	}

	statusCode := handlerCfg.StatusCode
	if statusCode == 0 {
		statusCode = 200
	}
	contentType := responseHeaders["Content-Type"]
	body := handlerCfg.Body

	requestHeaders := map[string]string{}
	for key, values := range ctx.Request.Header {
		requestHeaders[key] = strings.Join(values, ",")
	}

	vm := goja.New()
	ctxObject := vm.NewObject()
	requestObject := vm.NewObject()
	responseObject := vm.NewObject()
	requestHeadersObject := vm.NewObject()
	responseHeadersObject := vm.NewObject()

	for key, value := range requestHeaders {
		if err := requestHeadersObject.Set(key, value); err != nil {
			return err
		}
	}
	for key, value := range responseHeaders {
		if err := responseHeadersObject.Set(key, value); err != nil {
			return err
		}
	}

	if err := requestObject.Set("method", ctx.Method); err != nil {
		return err
	}
	if err := requestObject.Set("path", ctx.Path); err != nil {
		return err
	}
	if err := requestObject.Set("headers", requestHeadersObject); err != nil {
		return err
	}

	if err := responseObject.Set("status_code", statusCode); err != nil {
		return err
	}
	if err := responseObject.Set("content_type", contentType); err != nil {
		return err
	}
	if err := responseObject.Set("headers", responseHeadersObject); err != nil {
		return err
	}
	if err := responseObject.Set("body", body); err != nil {
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
	fetch := func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			panic(vm.ToValue("ctx.fetch requires url argument"))
		}
		url := call.Argument(0).String()
		res, err := ctx.Fetch().Get(url, nil).Execute()
		if err != nil {
			panic(vm.ToValue(err.Error()))
		}
		return vm.ToValue(res.String())
	}
	if err := ctxObject.Set("fetch", fetch); err != nil {
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

	if _, err := vm.RunString(handlerCfg.Script); err != nil {
		return err
	}

	for _, key := range responseHeadersObject.Keys() {
		ctx.Writer.Header().Set(key, responseHeadersObject.Get(key).String())
	}
	if contentTypeValue := responseObject.Get("content_type").String(); contentTypeValue != "" {
		ctx.Writer.Header().Set("Content-Type", contentTypeValue)
	}
	ctx.Status(int(responseObject.Get("status_code").ToInteger()))
	ctx.Write([]byte(responseObject.Get("body").String()))
	return nil
}
