package bootstrap

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-zoox/ingress/core/admin/config"
)

// seedAccessLogIfEmpty writes demo access lines when the configured log file is missing or empty.
func seedAccessLogIfEmpty(cfg *config.Config) error {
	if cfg == nil {
		return nil
	}
	path := strings.TrimSpace(cfg.AccessLogPath)
	if path == "" {
		return nil
	}
	if st, err := os.Stat(path); err == nil && st.Size() > 0 {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("seed access log mkdir: %w", err)
	}
	lines := generateSampleAccessLines(320)
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("seed access log write: %w", err)
	}
	return nil
}

type sampleRoute struct {
	host   string
	target string
	paths  []samplePath
	weight int
}

type samplePath struct {
	method string
	path   string
	status int
}

func generateSampleAccessLines(n int) []string {
	routes := []sampleRoute{
		{
			host: "api.example.com", target: "api.internal:8080", weight: 45,
			paths: []samplePath{
				{"GET", "/api/users", 200},
				{"GET", "/api/users/42", 200},
				{"POST", "/api/login", 401},
				{"GET", "/search", 200},
				{"GET", "/search", 400},
				{"POST", "/api/orders", 201},
				{"GET", "/v2/users", 200},
				{"GET", "/v2/health", 200},
				{"GET", "/public", 200},
			},
		},
		{
			host: "cdn.example.com", target: "minio.internal:9000", weight: 25,
			paths: []samplePath{
				{"GET", "/assets/app.js", 200},
				{"GET", "/assets/style.css", 200},
				{"GET", "/favicon.ico", 404},
			},
		},
		{
			host: "waf-demo.example.com", target: "httpbin.org:443", weight: 10,
			paths: []samplePath{
				{"GET", "/", 200},
				{"GET", "/admin", 403},
			},
		},
		{
			host: "admin.internal", target: "handler", weight: 8,
			paths: []samplePath{
				{"GET", "/healthz", 200},
			},
		},
	}

	ips := []string{
		"203.0.113.44", "198.51.100.8", "192.0.2.99", "203.0.113.12",
		"198.51.100.22", "203.0.113.88", "192.0.2.17", "10.0.0.5",
	}
	uas := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
		"curl/8.0",
		"ingress-admin/1.0",
		"Go-http-client/1.1",
	}

	end := time.Now()
	start := end.Add(-15 * 24 * time.Hour)
	rng := rand.New(rand.NewSource(99))

	totalWeight := 0
	for _, r := range routes {
		totalWeight += r.weight
	}

	lines := make([]string, 0, n)
	for i := 0; i < n; i++ {
		r := pickRoute(rng, routes, totalWeight)
		p := r.paths[rng.Intn(len(r.paths))]
		at := start.Add(time.Duration(i) * (end.Sub(start)) / time.Duration(n))
		at = at.Add(time.Duration(rng.Intn(60)) * time.Second)
		ms := 3 + rng.Intn(120)
		cacheHit := 0
		if rng.Float32() < 0.2 {
			cacheHit = 1
		}
		wafBlock := 0
		ip := ips[rng.Intn(len(ips))]
		lines = append(lines, fmt.Sprintf(
			`%s %s %s -> %s "%s %s HTTP/1.1" %d %dms cache_hit=%d waf_block=%d real_ip=%s referer=- ua=%s xff=%s tls_protocol=- tls_cipher=- upstream_status=%d upstream_response_length=1024 upstream_response_time=%dms`,
			at.Format("2006/01/02 15:04:05"),
			ip,
			r.host,
			r.target,
			p.method,
			p.path,
			p.status,
			ms,
			cacheHit,
			wafBlock,
			ip,
			uas[rng.Intn(len(uas))],
			ip,
			p.status,
			ms,
		))
	}
	return lines
}

func pickRoute(rng *rand.Rand, routes []sampleRoute, totalWeight int) sampleRoute {
	n := rng.Intn(totalWeight)
	for _, r := range routes {
		n -= r.weight
		if n < 0 {
			return r
		}
	}
	return routes[0]
}
