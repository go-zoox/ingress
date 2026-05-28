package core

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/ingress/core/service"
)

func TestBuildJSONKeyCanonicalLines_StableAcrossNoise(t *testing.T) {
	body := []byte(`{"product":{"id":42},"lang":"zh","timestamp":999,"trace":"x"}`)
	lines, ok := buildJSONKeyCanonicalLines(body, []string{"product.id", "lang"})
	if !ok {
		t.Fatal("expected ok")
	}
	body2 := []byte(`{"lang":"zh","product":{"id":42},"extra":1}`)
	lines2, ok := buildJSONKeyCanonicalLines(body2, []string{"lang", "product.id"})
	if !ok {
		t.Fatal("expected ok")
	}
	if strings.Join(lines, "\n") != strings.Join(lines2, "\n") {
		t.Fatalf("lines differ:\n%v\n%v", lines, lines2)
	}
}

func TestBuildJSONKeyCanonicalLines_MissingField(t *testing.T) {
	_, ok := buildJSONKeyCanonicalLines([]byte(`{"product":{}}`), []string{"product.id"})
	if ok {
		t.Fatal("expected missing product.id to fail")
	}
}

func TestBuildJSONKeyCanonicalLines_EmptyObject(t *testing.T) {
	_, ok := buildJSONKeyCanonicalLines([]byte(`{}`), []string{"product.id"})
	if ok {
		t.Fatal("expected empty object to fail")
	}
}

func TestBuildHTTPCacheStorageKey_V2PrefixWithJSONKey(t *testing.T) {
	pc := &httpCacheRuntime{
		KeyHash:      httpCacheKeyHashMD5,
		KeyHeaders:   []string{},
		MethodAllow:  map[string]struct{}{http.MethodPost: {}},
		KeyJSON:      []string{"product.id"},
		JSONKeyLines: []string{"jsonkey:product.id=7"},
	}
	req, _ := http.NewRequest(http.MethodPost, "http://example.com/api/x", nil)
	key := buildHTTPCacheStorageKey(req, "example.com", "/api/x", pc)
	if !strings.HasPrefix(key, httpCacheKeyPrefixV2) {
		t.Fatalf("key prefix: %s", key)
	}
}

func TestReadAndReplayRequestBody_ProxyCanReadAgain(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "http://x/", io.NopCloser(bytes.NewReader([]byte(`{"product":{"id":1}}`))))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := readAndReplayRequestBody(req, 4096); err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"product":{"id":1}}` {
		t.Fatalf("body=%q", b)
	}
}

func TestValidateConfig_POSTCachePathOK(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "api.example.com",
				Backend: rule.Backend{
					Cache: rule.BackendCache{
						Enabled: true,
						Default: cachePathDefaultBypass,
						Paths: []rule.BackendCachePathRule{
							{
								Match:   "/api/detail",
								Action:  cachePathActionCache,
								Methods: []string{"POST"},
								KeyJSON: []string{"product.id"},
							},
						},
					},
					Service: serviceStub(),
				},
			},
		},
	}
	if err := ValidateConfig(cfg); err != nil {
		t.Fatal(err)
	}
}

func TestValidateConfig_GlobalPOSTForbidden(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "api.example.com",
				Backend: rule.Backend{
					Cache: rule.BackendCache{
						Enabled: true,
						Methods: []string{"POST"},
					},
					Service: serviceStub(),
				},
			},
		},
	}
	if err := ValidateConfig(cfg); err == nil {
		t.Fatal("expected error for global POST in cache.methods")
	}
}

func TestValidateConfig_POSTWithoutKeyJSON(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "api.example.com",
				Backend: rule.Backend{
					Cache: rule.BackendCache{
						Enabled: true,
						Paths: []rule.BackendCachePathRule{
							{Match: "/x", Methods: []string{"POST"}, Action: cachePathActionCache},
						},
					},
					Service: serviceStub(),
				},
			},
		},
	}
	if err := ValidateConfig(cfg); err == nil {
		t.Fatal("expected error for POST without key_json")
	}
}

func serviceStub() service.Service {
	return service.Service{Name: "up", Port: 8080, Protocol: "http"}
}
