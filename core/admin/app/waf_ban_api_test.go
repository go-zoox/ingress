package app

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func doJSON(t *testing.T, client *http.Client, method, url, body string) (*http.Response, apiEnvelope) {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatal(err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	var env apiEnvelope
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &env); err != nil {
			t.Fatalf("unmarshal %q: %v", string(raw), err)
		}
	}
	return resp, env
}

func loginAsAdmin(t *testing.T, client *http.Client, base string) {
	t.Helper()
	resp, env := postJSON(t, client, base+"/api/v1/auth/login", `{"username":"admin","password":"admin"}`)
	if resp.StatusCode != http.StatusOK || env.Code != 200 {
		t.Fatalf("login failed status=%d env=%+v", resp.StatusCode, env)
	}
}

type banRowResp struct {
	ID      uint     `json:"id"`
	Kind    string   `json:"kind"`
	Value   string   `json:"value"`
	Scope   string   `json:"scope"`
	Actor   string   `json:"actor"`
	Enabled bool     `json:"enabled"`
	Active  bool     `json:"active"`
	Hits    int64    `json:"hits"`
	Hosts   []string `json:"hosts"`
}

func TestWAFBansAPI_CRUD(t *testing.T) {
	cfg, _ := testAdminConfig(t)
	ts, client := startTestAdminServer(t, cfg)
	loginAsAdmin(t, client, ts.URL)

	// Create
	resp, env := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/waf/bans",
		`{"kind":"ip","value":"203.0.113.42","reason":"sqlmap","ttl_seconds":3600}`)
	if resp.StatusCode != http.StatusOK || env.Code != 200 {
		t.Fatalf("create: status=%d env=%+v", resp.StatusCode, env)
	}
	var created banRowResp
	if err := json.Unmarshal(env.Result, &created); err != nil {
		t.Fatalf("unmarshal created: %v", err)
	}
	if created.Value != "203.0.113.42" || created.Kind != "ip" {
		t.Fatalf("unexpected created row: %+v", created)
	}
	if created.Actor == "" {
		t.Fatalf("actor should be resolved from the session, got empty")
	}

	// List
	resp, env = doJSON(t, client, http.MethodGet, ts.URL+"/api/v1/waf/bans?status=active", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list status=%d", resp.StatusCode)
	}
	var rows []banRowResp
	if err := json.Unmarshal(env.Result, &rows); err != nil {
		t.Fatalf("unmarshal list: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 active ban, got %d", len(rows))
	}

	// Get one
	resp, env = doJSON(t, client, http.MethodGet, ts.URL+"/api/v1/waf/bans/"+itoa(created.ID), "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get status=%d", resp.StatusCode)
	}

	// Patch TTL (renew)
	resp, env = doJSON(t, client, http.MethodPatch, ts.URL+"/api/v1/waf/bans/"+itoa(created.ID),
		`{"ttl_seconds":7200}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch status=%d env=%+v", resp.StatusCode, env)
	}

	// Delete
	resp, env = doJSON(t, client, http.MethodDelete, ts.URL+"/api/v1/waf/bans/"+itoa(created.ID), "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete status=%d env=%+v", resp.StatusCode, env)
	}

	resp, env = doJSON(t, client, http.MethodGet, ts.URL+"/api/v1/waf/bans?status=active", "")
	var after []banRowResp
	_ = json.Unmarshal(env.Result, &after)
	if len(after) != 0 {
		t.Fatalf("expected no active bans after delete, got %d", len(after))
	}
}

func TestWAFBansAPI_ValidationErrors(t *testing.T) {
	cfg, _ := testAdminConfig(t)
	ts, client := startTestAdminServer(t, cfg)
	loginAsAdmin(t, client, ts.URL)

	cases := []struct {
		name string
		body string
		want int
		msg  string
	}{
		{
			name: "invalid ip",
			body: `{"kind":"ip","value":"999.1.1.1","ttl_seconds":60}`,
			want: http.StatusBadRequest,
			msg:  "invalid_value:",
		},
		{
			name: "invalid regex",
			body: `{"kind":"ua_regex","value":"(?i","ttl_seconds":60}`,
			want: http.StatusBadRequest,
			msg:  "invalid_value:",
		},
		{
			name: "invalid kind",
			body: `{"kind":"nope","value":"x","ttl_seconds":60}`,
			want: http.StatusBadRequest,
			msg:  "invalid_kind:",
		},
		{
			name: "permanent without confirmation",
			body: `{"kind":"ip","value":"203.0.113.42"}`,
			want: http.StatusBadRequest,
			msg:  "confirm_required:",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, env := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/waf/bans", tc.body)
			if resp.StatusCode != tc.want {
				t.Fatalf("status=%d want %d (env=%+v)", resp.StatusCode, tc.want, env)
			}
			if !strings.HasPrefix(env.Message, tc.msg) {
				t.Fatalf("message=%q want prefix %q", env.Message, tc.msg)
			}
		})
	}
}

func TestWAFBansAPI_DuplicateAndNotFound(t *testing.T) {
	cfg, _ := testAdminConfig(t)
	ts, client := startTestAdminServer(t, cfg)
	loginAsAdmin(t, client, ts.URL)

	body := `{"kind":"ip","value":"198.51.100.7","ttl_seconds":3600}`
	if resp, env := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/waf/bans", body); resp.StatusCode != http.StatusOK {
		t.Fatalf("first create: %d %+v", resp.StatusCode, env)
	}

	resp, env := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/waf/bans", body)
	if resp.StatusCode != http.StatusBadRequest || !strings.HasPrefix(env.Message, "duplicate:") {
		t.Fatalf("duplicate: status=%d message=%q", resp.StatusCode, env.Message)
	}

	resp, env = doJSON(t, client, http.MethodDelete, ts.URL+"/api/v1/waf/bans/99999", "")
	if resp.StatusCode != http.StatusNotFound || !strings.HasPrefix(env.Message, "not_found:") {
		t.Fatalf("not found: status=%d message=%q", resp.StatusCode, env.Message)
	}
}

func TestWAFBansAPI_Preview(t *testing.T) {
	cfg, _ := testAdminConfig(t)
	ts, client := startTestAdminServer(t, cfg)
	loginAsAdmin(t, client, ts.URL)

	resp, env := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/waf/bans/preview",
		`{"kind":"ip","value":"203.0.113.42"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("preview status=%d env=%+v", resp.StatusCode, env)
	}
	var preview struct {
		Kind       string   `json:"kind"`
		Normalized string   `json:"normalized"`
		Hits24h    int64    `json:"hits_24h"`
		Warnings   []string `json:"warnings"`
		Blocked    bool     `json:"blocked"`
	}
	if err := json.Unmarshal(env.Result, &preview); err != nil {
		t.Fatalf("unmarshal preview: %v", err)
	}
	if preview.Kind != "ip" || preview.Normalized != "203.0.113.42" {
		t.Fatalf("unexpected preview: %+v", preview)
	}
	if preview.Blocked {
		t.Fatalf("a public IP should not be blocked")
	}
}

func TestWAFBansAPI_UARegexWarning(t *testing.T) {
	cfg, _ := testAdminConfig(t)
	ts, client := startTestAdminServer(t, cfg)
	loginAsAdmin(t, client, ts.URL)

	_, env := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/waf/bans/preview",
		`{"kind":"ua_contains","value":"sqlmap"}`)
	var preview struct {
		Warnings []string `json:"warnings"`
	}
	if err := json.Unmarshal(env.Result, &preview); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	found := false
	for _, w := range preview.Warnings {
		if strings.Contains(w, "UA 是客户端可控字段") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected UA bypass warning, got %v", preview.Warnings)
	}
}

func TestWAFBansAPI_RequiresAuth(t *testing.T) {
	cfg, _ := testAdminConfig(t)
	ts, client := startTestAdminServer(t, cfg)
	// No login.
	resp, _ := doJSON(t, client, http.MethodGet, ts.URL+"/api/v1/waf/bans", "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth, got %d", resp.StatusCode)
	}
}

func itoa(v uint) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
