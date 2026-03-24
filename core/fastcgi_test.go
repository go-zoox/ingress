package core

import (
	"net/http/httptest"
	"testing"

	"github.com/go-zoox/ingress/core/service"
	"github.com/yookoala/gofast"
)

func TestResolveFastCGIScriptPath(t *testing.T) {
	tests := []struct {
		name     string
		service  *service.Service
		expected string
	}{
		{
			name: "default entry script",
			service: &service.Service{
				FastCGI: service.FastCGI{},
			},
			expected: "/index.php",
		},
		{
			name: "thinkphp default entry script",
			service: &service.Service{
				FastCGI: service.FastCGI{
					Framework: "thinkphp",
				},
			},
			expected: "/public/index.php",
		},
		{
			name: "custom script keeps value",
			service: &service.Service{
				FastCGI: service.FastCGI{
					Framework: "thinkphp",
					Script: service.Script{
						Filename: "app.php",
					},
				},
			},
			expected: "/app.php",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveFastCGIScriptPath(tt.service)
			if got != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestApplyThinkPHPParams(t *testing.T) {
	req := gofast.NewRequest(httptest.NewRequest("GET", "http://example.com/blog/post?id=1", nil))
	svc := &service.Service{
		FastCGI: service.FastCGI{
			Framework: "thinkphp",
		},
	}

	applyFrameworkFastCGIParams(req, svc, "/public/index.php")

	if req.Params["DOCUMENT_ROOT"] != "/var/www/html" {
		t.Fatalf("expected DOCUMENT_ROOT=/var/www/html, got %s", req.Params["DOCUMENT_ROOT"])
	}
	if req.Params["SCRIPT_NAME"] != "/public/index.php" {
		t.Fatalf("expected SCRIPT_NAME=/public/index.php, got %s", req.Params["SCRIPT_NAME"])
	}
	if req.Params["SCRIPT_FILENAME"] != "/var/www/html/public/index.php" {
		t.Fatalf("expected SCRIPT_FILENAME=/var/www/html/public/index.php, got %s", req.Params["SCRIPT_FILENAME"])
	}
	if req.Params["PATH_INFO"] != "/blog/post" {
		t.Fatalf("expected PATH_INFO=/blog/post, got %s", req.Params["PATH_INFO"])
	}
	if req.Params["REQUEST_URI"] != "/blog/post?id=1" {
		t.Fatalf("expected REQUEST_URI=/blog/post?id=1, got %s", req.Params["REQUEST_URI"])
	}
}
