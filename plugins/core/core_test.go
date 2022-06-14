package core

import (
	"testing"

	ingress "github.com/go-zoox/ingress/core"
)

func TestGetConfig(t *testing.T) {
	config := &ingress.Config{
		Port: 8080,
		Rules: []ingress.Rule{
			{
				Host: "httpbin.example.com",
				Backend: ingress.Backend{
					ServiceName: "httpbin",
					ServicePort: 8080,
				},
				Paths: []ingress.Path{
					{
						Path: "/get",
						Backend: ingress.Backend{
							ServiceName: "httpbin2",
							ServicePort: 8082,
						},
					},
					{
						Path: "^/api/(.*)",
						Backend: ingress.Backend{
							ServiceName: "httpbin3",
							ServicePort: 8083,
						},
					},
				},
			},
			{
				Host: "omg.example.com",
				Backend: ingress.Backend{
					ServiceName: "omg",
					ServicePort: 8080,
				},
			},
		},
	}

	c := &Core{
		Application: &ingress.Application{
			Config: config,
		},
	}

	// 0. host not match
	if cfg := c.getConfig("notfound.example.com", "/none-path-matched"); cfg.ServiceName != "" || cfg.ServicePort != 0 {
		t.Errorf("getConfig(httpbin.example.com, /none-path-matched) = %s:%d, want empty", cfg.ServiceName, cfg.ServicePort)
	}

	// 1. host match
	if cfg := c.getConfig("httpbin.example.com", "/none-path-matched"); cfg.ServiceName != "httpbin" || cfg.ServicePort != 8080 {
		t.Errorf("getConfig(httpbin.example.com, /none-path-matched) = %s:%d, want httpbin:8080", cfg.ServiceName, cfg.ServicePort)
	}

	if cfg := c.getConfig("omg.example.com", "/get"); cfg.ServiceName != "omg" || cfg.ServicePort != 8080 {
		t.Errorf("getConfig(omg.example.com, /get) = %s:%d, want omg:8080", cfg.ServiceName, cfg.ServicePort)
	}

	// 2. path match
	if cfg := c.getConfig("httpbin.example.com", "/get"); cfg.ServiceName != "httpbin2" || cfg.ServicePort != 8082 {
		t.Errorf("getConfig(httpbin.example.com, /get) = %s:%d, want httpbin2:8082", cfg.ServiceName, cfg.ServicePort)
	}

	// 3. regexp match
	if cfg := c.getConfig("httpbin.example.com", "/api"); cfg.ServiceName != "httpbin" || cfg.ServicePort != 8080 {
		t.Errorf("getConfig(httpbin.example.com, /api/get) = %s:%d, want httpbin:8080", cfg.ServiceName, cfg.ServicePort)
	}

	// 4. (not recommand) /get will match any path contains /get, such as /api/get, /xxx/get, /yyy/get/xxx
	if cfg := c.getConfig("httpbin.example.com", "/api/get"); cfg.ServiceName != "httpbin2" || cfg.ServicePort != 8082 {
		t.Errorf("getConfig(httpbin.example.com, /api/get) = %s:%d, want httpbin2:8082", cfg.ServiceName, cfg.ServicePort)
	}

	// 5. regexp match, same as 3
	if cfg := c.getConfig("httpbin.example.com", "/api/post"); cfg.ServiceName != "httpbin3" || cfg.ServicePort != 8083 {
		t.Errorf("getConfig(httpbin.example.com, /api/get) = %s:%d, want httpbin3:8083", cfg.ServiceName, cfg.ServicePort)
	}
}
