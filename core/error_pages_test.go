package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompileErrorPages_BuiltinDefaults(t *testing.T) {
	pages, err := compileErrorPages(&Config{})
	if err != nil {
		t.Fatal(err)
	}
	html := pages.Render(404, false, ErrorPageDetail{Hostname: "secret.example", Path: "/x"})
	if strings.Contains(html, "secret.example") {
		t.Fatal("safe mode must not echo host")
	}
	if !strings.Contains(html, "Not Found") {
		t.Fatal("expected builtin 404 title")
	}
}

func TestCompileErrorPages_CustomFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "401.html")
	const custom = "<html><body>custom 401</body></html>"
	if err := os.WriteFile(path, []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}
	pages, err := compileErrorPages(&Config{
		ErrorPages: ErrorPages{
			Pages: map[string]ErrorPageSpec{
				"401": {Type: "file", File: path},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := pages.Render(401, true, ErrorPageDetail{}); got != custom {
		t.Fatalf("got %q", got)
	}
}

func TestCompileErrorPages_CustomInline(t *testing.T) {
	pages, err := compileErrorPages(&Config{
		ErrorPages: ErrorPages{
			Pages: map[string]ErrorPageSpec{
				"502": {Type: "inline", Body: "<p>bad gateway</p>"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := pages.Render(502, false, ErrorPageDetail{}); got != "<p>bad gateway</p>" {
		t.Fatalf("got %q", got)
	}
}

func TestValidateErrorPages_UnsupportedStatus(t *testing.T) {
	err := validateErrorPages(&Config{
		ErrorPages: ErrorPages{
			Pages: map[string]ErrorPageSpec{
				"418": {Type: "builtin"},
			},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "418") {
		t.Fatalf("expected unsupported status error, got %v", err)
	}
}

func TestShouldUseWAFErrorPage(t *testing.T) {
	if !shouldUseWAFErrorPage(403, "text/plain; charset=utf-8", "Forbidden\n") {
		t.Fatal("expected default WAF block to use error page")
	}
	if shouldUseWAFErrorPage(403, "application/json", `{"ok":false}`) {
		t.Fatal("custom WAF content type should keep WAF body")
	}
}

func TestBuiltinErrorPageCopy_AllSupported(t *testing.T) {
	for _, status := range supportedErrorPageStatuses {
		title, subtitle := builtinErrorPageCopy(status)
		if title == "" || subtitle == "" {
			t.Fatalf("missing copy for %d", status)
		}
	}
}
