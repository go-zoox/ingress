package jobs

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	ingjobs "github.com/go-zoox/ingress/core/jobs"
	"github.com/go-zoox/ingress/core/jobs/scriptexec"
)

type execEnv struct {
	timeoutSec int64
	policy     ingjobs.AdminJobs
}

func maxCaptureBytes(policy ingjobs.AdminJobs) int64 {
	if policy.CommandMaxOutputBytes > 0 {
		return policy.CommandMaxOutputBytes
	}
	return 65536
}

func runHTTPCall(ctx context.Context, params ingjobs.JobParams, timeoutSec int64, policy ingjobs.AdminJobs) (ExecOutcome, error) {
	method := strings.ToUpper(strings.TrimSpace(params.Method))
	if method == "" {
		method = http.MethodGet
	}
	url := strings.TrimSpace(params.URL)
	if url == "" {
		return ExecOutcome{}, fmt.Errorf("http_call: url is required")
	}

	req, err := http.NewRequestWithContext(ctx, method, url, strings.NewReader(params.Body))
	if err != nil {
		return ExecOutcome{}, err
	}
	for k, v := range params.Headers {
		req.Header.Set(k, v)
	}
	if params.Body != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{
		Timeout: time.Duration(DefaultTimeoutSec(timeoutSec)) * time.Second,
	}
	if params.InsecureTLS {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		}
	}

	res, err := client.Do(req)
	if err != nil {
		return ExecOutcome{}, err
	}
	defer res.Body.Close()

	maxBody := maxCaptureBytes(policy)
	body, _ := io.ReadAll(io.LimitReader(res.Body, maxBody))
	bodyStr := string(body)
	if int64(len(body)) >= maxBody {
		bodyStr += "\n...(truncated)"
	}

	headers := make(map[string]string, len(res.Header))
	for k, vals := range res.Header {
		headers[k] = strings.Join(vals, ", ")
	}

	detail := RunResultDetail{
		HTTP: &HTTPRunResult{
			StatusCode: res.StatusCode,
			Headers:    headers,
			Body:       bodyStr,
		},
	}
	preview := fmt.Sprintf("HTTP %d", res.StatusCode)
	out := ExecOutcome{Preview: preview, Detail: detail}

	expect := params.ExpectStatus
	if len(expect) == 0 {
		if res.StatusCode >= 200 && res.StatusCode < 300 {
			return out, nil
		}
		return out, fmt.Errorf("unexpected status %d", res.StatusCode)
	}
	for _, code := range expect {
		if res.StatusCode == int(code) {
			return out, nil
		}
	}
	return out, fmt.Errorf("unexpected status %d (expected %v)", res.StatusCode, expect)
}

func runCommand(ctx context.Context, params ingjobs.JobParams, env execEnv) (ExecOutcome, error) {
	if !env.policy.CommandExecutionEnabled() {
		return ExecOutcome{}, fmt.Errorf("command execution is disabled (admin.jobs.allow_command: false)")
	}
	engine := ingjobs.NormalizeScriptEngine(params.Engine)
	if ingjobs.IsEmbeddedScriptEngine(engine) {
		return runEmbeddedScript(ctx, params, env, engine)
	}

	script := ingjobs.ScriptContent(params)
	if strings.TrimSpace(script) == "" {
		return ExecOutcome{}, fmt.Errorf("script: content is required")
	}
	shellPath, err := ingjobs.ResolveShellExecutable(ingjobs.ScriptShellName(params))
	if err != nil {
		return ExecOutcome{}, err
	}
	if len(env.policy.CommandAllowlist) > 0 && !commandAllowed(shellPath, env.policy.CommandAllowlist) {
		return ExecOutcome{}, fmt.Errorf("shell %q is not allowlisted", shellPath)
	}

	workdir := strings.TrimSpace(params.Workdir)
	if workdir == "" {
		workdir = strings.TrimSpace(env.policy.CommandWorkdir)
	}

	maxOut := maxCaptureBytes(env.policy)

	cmd := exec.CommandContext(ctx, shellPath, "-c", script)
	if workdir != "" {
		cmd.Dir = workdir
	}
	if len(params.Env) > 0 {
		cmd.Env = append(cmd.Environ(), mapEnv(params.Env)...)
	}

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	runErr := cmd.Run()
	log := truncateOutput(buf.String(), maxOut)
	detail := RunResultDetail{Command: &CommandRunResult{Log: log}}
	out := ExecOutcome{Preview: truncateOutput(log, 120), Detail: detail}
	if runErr != nil {
		return out, runErr
	}
	return out, nil
}

func runEmbeddedScript(ctx context.Context, params ingjobs.JobParams, env execEnv, engine string) (ExecOutcome, error) {
	script := ingjobs.ScriptContent(params)
	if strings.TrimSpace(script) == "" {
		return ExecOutcome{}, fmt.Errorf("script: content is required")
	}
	timeout := time.Duration(DefaultTimeoutSec(env.timeoutSec)) * time.Second
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	maxOut := maxCaptureBytes(env.policy)
	log, err := scriptexec.Run(runCtx, engine, script, scriptexec.Options{
		MaxOutputBytes: maxOut,
		Workdir:        params.Workdir,
		Env:            params.Env,
	})
	detail := RunResultDetail{Command: &CommandRunResult{Log: log}}
	out := ExecOutcome{Preview: truncateOutput(log, 120), Detail: detail}
	if err != nil {
		return out, err
	}
	return out, nil
}

func mapEnv(env map[string]string) []string {
	out := make([]string, 0, len(env))
	for k, v := range env {
		out = append(out, k+"="+v)
	}
	return out
}

func truncateOutput(s string, max int64) string {
	if max <= 0 || int64(len(s)) <= max {
		return s
	}
	return s[:max] + "\n...(truncated)"
}

func messageOutcome(msg string, err error) (ExecOutcome, error) {
	return ExecOutcome{Preview: msg, Detail: RunResultDetail{Message: msg}}, err
}
