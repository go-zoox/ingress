package scriptexec

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dop251/goja"
)

func runJavaScript(ctx context.Context, script string, opts Options) (string, error) {
	var buf bytes.Buffer
	maxOut := optsMaxOutput(opts.MaxOutputBytes)

	vm := goja.New()
	console := vm.NewObject()
	logLine := func(call goja.FunctionCall) goja.Value {
		parts := make([]string, len(call.Arguments))
		for i, arg := range call.Arguments {
			parts[i] = arg.String()
		}
		line := strings.Join(parts, " ")
		if buf.Len() > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString(line)
		if int64(buf.Len()) > maxOut {
			return goja.Undefined()
		}
		return goja.Undefined()
	}
	if err := console.Set("log", logLine); err != nil {
		return "", err
	}
	if err := console.Set("error", logLine); err != nil {
		return "", err
	}
	if err := console.Set("warn", logLine); err != nil {
		return "", err
	}
	if err := vm.Set("console", console); err != nil {
		return "", err
	}

	fetchFn := func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			panic(vm.ToValue("fetch requires url"))
		}
		url := call.Argument(0).String()
		method := http.MethodGet
		var body io.Reader
		if len(call.Arguments) > 1 {
			opt := call.Argument(1)
			if !goja.IsUndefined(opt) && !goja.IsNull(opt) {
				if m := opt.ToObject(vm).Get("method"); m != nil && !goja.IsUndefined(m) {
					method = strings.ToUpper(m.String())
				}
				if b := opt.ToObject(vm).Get("body"); b != nil && !goja.IsUndefined(b) {
					body = strings.NewReader(b.String())
				}
			}
		}
		req, err := http.NewRequestWithContext(ctx, method, url, body)
		if err != nil {
			panic(vm.ToValue(err.Error()))
		}
		if body != nil && req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", "application/json")
		}
		client := &http.Client{Timeout: 60 * time.Second}
		res, err := client.Do(req)
		if err != nil {
			panic(vm.ToValue(err.Error()))
		}
		defer res.Body.Close()
		resBody, _ := io.ReadAll(io.LimitReader(res.Body, maxOut))
		response := vm.NewObject()
		headers := vm.NewObject()
		for key, vals := range res.Header {
			if len(vals) > 0 {
				_ = headers.Set(key, vals[0])
			}
		}
		bodyStr := string(resBody)
		_ = response.Set("status", res.StatusCode)
		_ = response.Set("ok", res.StatusCode >= 200 && res.StatusCode < 300)
		_ = response.Set("headers", headers)
		_ = response.Set("text", func(goja.FunctionCall) goja.Value {
			return vm.ToValue(bodyStr)
		})
		_ = response.Set("json", func(goja.FunctionCall) goja.Value {
			var parsed any
			if err := json.Unmarshal(resBody, &parsed); err != nil {
				panic(vm.ToValue(err.Error()))
			}
			return vm.ToValue(parsed)
		})
		return response
	}
	if err := vm.Set("fetch", fetchFn); err != nil {
		return "", err
	}

	scriptWrapper := fmt.Sprintf("(async () => {\n%s\n})()", script)
	result, err := vm.RunString(scriptWrapper)
	if err != nil {
		return truncateLog(buf.String(), maxOut), err
	}
	exported := result.Export()
	promise, ok := exported.(*goja.Promise)
	if !ok {
		return truncateLog(buf.String(), maxOut), nil
	}
	switch promise.State() {
	case goja.PromiseStateFulfilled:
		return truncateLog(buf.String(), maxOut), nil
	case goja.PromiseStateRejected:
		return truncateLog(buf.String(), maxOut), errors.New(promise.Result().String())
	case goja.PromiseStatePending:
		return truncateLog(buf.String(), maxOut), errors.New("javascript script is still pending")
	default:
		return truncateLog(buf.String(), maxOut), errors.New("javascript script ended in unknown promise state")
	}
}

func optsMaxOutput(max int64) int64 {
	if max > 0 {
		return max
	}
	return 65536
}

func truncateLog(s string, max int64) string {
	if max <= 0 || int64(len(s)) <= max {
		return s
	}
	return s[:max] + "\n...(truncated)"
}
