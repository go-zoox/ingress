package jobs

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ingjobs "github.com/go-zoox/ingress/core/jobs"
)

func TestRunHTTPCall_SuccessCapturesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q", r.Method)
		}
		if got := r.Header.Get("X-Probe"); got != "ingress" {
			t.Fatalf("X-Probe = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Resp", "ok")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"status":"created"}`))
	}))
	t.Cleanup(srv.Close)

	out, err := runHTTPCall(context.Background(), ingjobs.JobParams{
		Method:  "POST",
		URL:     srv.URL + "/hook",
		Headers: map[string]string{"X-Probe": "ingress"},
		Body:    `{"ping":1}`,
	}, 5, ingjobs.AdminJobs{})
	if err != nil {
		t.Fatal(err)
	}
	if out.Preview != "HTTP 201" {
		t.Fatalf("preview = %q", out.Preview)
	}
	if out.Detail.HTTP == nil {
		t.Fatal("expected http detail")
	}
	if out.Detail.HTTP.StatusCode != 201 {
		t.Fatalf("status = %d", out.Detail.HTTP.StatusCode)
	}
	if out.Detail.HTTP.Headers["Content-Type"] != "application/json" {
		t.Fatalf("content-type = %q", out.Detail.HTTP.Headers["Content-Type"])
	}
	if !strings.Contains(out.Detail.HTTP.Body, `"status":"created"`) {
		t.Fatalf("body = %q", out.Detail.HTTP.Body)
	}
}

func TestRunHTTPCall_UnexpectedStatusFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	t.Cleanup(srv.Close)

	out, err := runHTTPCall(context.Background(), ingjobs.JobParams{
		URL: srv.URL,
	}, 5, ingjobs.AdminJobs{})
	if err == nil {
		t.Fatal("expected error")
	}
	if out.Detail.HTTP == nil || out.Detail.HTTP.StatusCode != 500 {
		t.Fatalf("detail = %+v", out.Detail.HTTP)
	}
}

func TestRunHTTPCall_ExpectStatusList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(srv.Close)

	_, err := runHTTPCall(context.Background(), ingjobs.JobParams{
		URL:          srv.URL,
		ExpectStatus: []int64{200},
	}, 5, ingjobs.AdminJobs{})
	if err == nil {
		t.Fatal("expected mismatch error")
	}

	out, err := runHTTPCall(context.Background(), ingjobs.JobParams{
		URL:          srv.URL,
		ExpectStatus: []int64{202},
	}, 5, ingjobs.AdminJobs{})
	if err != nil {
		t.Fatal(err)
	}
	if out.Detail.HTTP.StatusCode != 202 {
		t.Fatalf("status = %d", out.Detail.HTTP.StatusCode)
	}
}

func TestRunHTTPCall_MissingURL(t *testing.T) {
	_, err := runHTTPCall(context.Background(), ingjobs.JobParams{}, 5, ingjobs.AdminJobs{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunCommand_Success(t *testing.T) {
	policy := ingjobs.AdminJobs{CommandMaxOutputBytes: 4096}
	out, err := runCommand(context.Background(), ingjobs.JobParams{
		Script: "echo hello jobs",
		Shell:  "sh",
	}, execEnv{timeoutSec: 5, policy: policy})
	if err != nil {
		t.Fatal(err)
	}
	if out.Detail.Command == nil || !strings.Contains(out.Detail.Command.Log, "hello jobs") {
		t.Fatalf("log = %+v", out.Detail.Command)
	}
}

func TestRunCommand_LegacyCommandArgs(t *testing.T) {
	policy := ingjobs.AdminJobs{CommandMaxOutputBytes: 4096}
	out, err := runCommand(context.Background(), ingjobs.JobParams{
		Command: "/bin/echo",
		Args:    []string{"legacy"},
	}, execEnv{timeoutSec: 5, policy: policy})
	if err != nil {
		t.Fatal(err)
	}
	if out.Detail.Command == nil || !strings.Contains(out.Detail.Command.Log, "legacy") {
		t.Fatalf("log = %+v", out.Detail.Command)
	}
}

func TestRunCommand_DisabledWhenPolicyOff(t *testing.T) {
	disabled := false
	_, err := runCommand(context.Background(), ingjobs.JobParams{Command: "/bin/echo"}, execEnv{
		timeoutSec: 5,
		policy:     ingjobs.AdminJobs{AllowCommand: &disabled},
	})
	if err == nil || !strings.Contains(err.Error(), "allow_command") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunCommand_NotAllowlisted(t *testing.T) {
	_, err := runCommand(context.Background(), ingjobs.JobParams{
		Script: "echo nope",
		Shell:  "sh",
	}, execEnv{
		timeoutSec: 5,
		policy: ingjobs.AdminJobs{
			CommandAllowlist: []string{"/bin/bash"},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "allowlisted") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunCommand_NonZeroExitFails(t *testing.T) {
	policy := ingjobs.AdminJobs{}
	out, err := runCommand(context.Background(), ingjobs.JobParams{
		Engine: ingjobs.ScriptEngineShell,
		Script: "echo oops >&2; exit 2",
		Shell:  "sh",
	}, execEnv{timeoutSec: 5, policy: policy})
	if err == nil {
		t.Fatal("expected exit error")
	}
	if out.Detail.Command == nil || !strings.Contains(out.Detail.Command.Log, "oops") {
		t.Fatalf("log = %+v", out.Detail.Command)
	}
}

func TestRunCommand_JavaScriptEmbedded(t *testing.T) {
	out, err := runCommand(context.Background(), ingjobs.JobParams{
		Engine: ingjobs.ScriptEngineJavaScript,
		Script: `console.log("embedded js")`,
	}, execEnv{timeoutSec: 5, policy: ingjobs.AdminJobs{}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.Detail.Command.Log, "embedded js") {
		t.Fatalf("log = %q", out.Detail.Command.Log)
	}
}

func TestRunCommand_GoEmbedded(t *testing.T) {
	out, err := runCommand(context.Background(), ingjobs.JobParams{
		Engine: ingjobs.ScriptEngineGo,
		Script: `import "fmt"
fmt.Println("embedded go")`,
	}, execEnv{timeoutSec: 5, policy: ingjobs.AdminJobs{}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.Detail.Command.Log, "embedded go") {
		t.Fatalf("log = %q", out.Detail.Command.Log)
	}
}

func TestRunCommand_EmbeddedSkipsAllowlist(t *testing.T) {
	policy := ingjobs.AdminJobs{CommandAllowlist: []string{"/bin/bash"}}
	out, err := runCommand(context.Background(), ingjobs.JobParams{
		Engine: ingjobs.ScriptEngineJavaScript,
		Script: `console.log("no allowlist")`,
	}, execEnv{timeoutSec: 5, policy: policy})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.Detail.Command.Log, "no allowlist") {
		t.Fatalf("log = %q", out.Detail.Command.Log)
	}
}

func TestRunCommand_ShellEchoBuiltin(t *testing.T) {
	policy := ingjobs.AdminJobs{CommandAllowlist: []string{"/bin/sh"}}
	out, err := runCommand(context.Background(), ingjobs.JobParams{
		Engine: ingjobs.ScriptEngineShell,
		Shell:  "sh",
		Script: `echo hello jobs`,
	}, execEnv{timeoutSec: 5, policy: policy})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.Detail.Command.Log, "hello jobs") {
		t.Fatalf("log = %q", out.Detail.Command.Log)
	}
}

func TestTruncateOutput(t *testing.T) {
	if got := truncateOutput("abcdef", 3); got != "abc\n...(truncated)" {
		t.Fatalf("got %q", got)
	}
	if got := truncateOutput("ab", 5); got != "ab" {
		t.Fatalf("got %q", got)
	}
}
