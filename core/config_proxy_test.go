package core

import (
	"os"
	"testing"

	"github.com/go-zoox/config"
)

func TestConfigProxyTrustProxyLoad(t *testing.T) {
	var cfg Config
	raw := []byte(`
port: 8080
proxy:
  trust_proxy: true
rules:
  - host: example.com
    backend:
      service:
        name: backend
        port: 8080
`)

	tmp, err := os.CreateTemp("", "ingress-proxy-trust-*.yaml")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		t.Fatalf("write temp file: %v", err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatalf("close temp file: %v", err)
	}

	if err := config.Load(&cfg, &config.LoadOptions{FilePath: tmp.Name()}); err != nil {
		t.Fatalf("load config: %v", err)
	}

	if !cfg.Proxy.TrustProxy {
		t.Fatalf("expected proxy.trust_proxy=true, got false")
	}
}
