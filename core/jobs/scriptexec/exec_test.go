package scriptexec

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRun_JavaScriptConsole(t *testing.T) {
	log, err := Run(context.Background(), "javascript", `console.log("hello", "jobs")`, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(log, "hello jobs") {
		t.Fatalf("log = %q", log)
	}
}

func TestRun_JavaScriptAsync(t *testing.T) {
	_, err := Run(context.Background(), "javascript", `await Promise.resolve(); throw new Error("boom")`, Options{})
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("err = %v", err)
	}
}

func TestRun_GoFmtPrintln(t *testing.T) {
	log, err := Run(context.Background(), "go", `import "fmt"
fmt.Println("go", "jobs")`, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(log, "go jobs") {
		t.Fatalf("log = %q", log)
	}
}

func TestRun_GoImportBlockStdlib(t *testing.T) {
	log, err := Run(context.Background(), "go", `import (
	"fmt"
	"strings"
)
fmt.Println(strings.ToUpper("jobs"))`, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(log, "JOBS") {
		t.Fatalf("log = %q", log)
	}
}

func TestRun_JavaScriptFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)

	script := `console.log("fetching")
const res = await fetch("` + srv.URL + `")
const data = await res.json()
console.log("status", res.status, "ok", data.ok)`
	log, err := Run(context.Background(), "javascript", script, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(log, "fetching") || !strings.Contains(log, "status 200") || !strings.Contains(log, "ok true") {
		t.Fatalf("log = %q", log)
	}
}

func TestRun_OutputTruncated(t *testing.T) {
	log, err := Run(context.Background(), "javascript", `console.log("x".repeat(100))`, Options{MaxOutputBytes: 10})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(log, "...(truncated)") {
		t.Fatalf("log = %q", log)
	}
}

func TestRun_UnsupportedEngine(t *testing.T) {
	_, err := Run(context.Background(), "python", `print(1)`, Options{})
	if err == nil {
		t.Fatal("expected error")
	}
}
