package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	admincfg "github.com/go-zoox/ingress/core/admin/config"
)

func TestAnalyzeRouteImpacts_addedAndChanged(t *testing.T) {
	published := `
version: v1
port: 8080
rules:
  - host: a.example.com
    backend:
      type: service
      service:
        name: upstream-a
        port: 80
`
	draft := `
version: v1
port: 8080
rules:
  - host: a.example.com
    backend:
      type: service
      service:
        name: upstream-b
        port: 80
  - host: b.example.com
    paths:
      - path: /v2
        backend:
          type: service
          service:
            name: upstream-c
            port: 80
    backend:
      type: service
      service:
        name: upstream-c-root
        port: 80
`
	ing := NewIngress(testIngressAdminCfg(t))
	impacts, err := AnalyzeRouteImpacts(ing, published, draft)
	if err != nil {
		t.Fatal(err)
	}
	if len(impacts) < 2 {
		t.Fatalf("expected at least 2 impacts, got %d: %+v", len(impacts), impacts)
	}
	var hasChanged, hasAdded bool
	for _, im := range impacts {
		switch im.Kind {
		case "changed":
			hasChanged = true
			if im.Host != "a.example.com" {
				t.Fatalf("changed host: %+v", im)
			}
		case "added":
			hasAdded = true
			if im.Host != "b.example.com" || !strings.HasPrefix(im.Path, "/") {
				t.Fatalf("added: %+v", im)
			}
		}
	}
	if !hasChanged || !hasAdded {
		t.Fatalf("impacts=%+v", impacts)
	}
}

func testIngressAdminCfg(t *testing.T) *admincfg.Config {
	t.Helper()
	path := filepath.Join("..", "..", "..", "examples", "admin-console", "ingress.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Skip("sample ingress not found")
	}
	return &admincfg.Config{IngressConfigPath: path}
}

func TestGlobalTouchesFromModules(t *testing.T) {
	got := globalTouchesFromModules([]string{"rules", "waf", "https"})
	if len(got) != 2 {
		t.Fatalf("got %v", got)
	}
}
