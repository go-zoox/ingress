package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	gozooxoauth2 "github.com/go-zoox/oauth2"
	"github.com/go-zoox/zoox"
)

// ---------------------------------------------------------------------------
// callback path detection (custom redirect_url support)
// ---------------------------------------------------------------------------

func TestOAuth2CallbackPathFor(t *testing.T) {
	cases := []struct {
		name        string
		redirectURL string
		want        string
	}{
		{name: "empty uses default", redirectURL: "", want: oauth2CallbackPath},
		{name: "default path", redirectURL: "https://example.com/oauth2/callback", want: "/oauth2/callback"},
		{name: "custom provider convention", redirectURL: "https://example.com/login/doreamon/callback", want: "/login/doreamon/callback"},
		{name: "host only", redirectURL: "https://example.com", want: oauth2CallbackPath},
		{name: "root path", redirectURL: "https://example.com/", want: oauth2CallbackPath},
		{name: "query and fragment ignored", redirectURL: "https://example.com/login/cb?x=1#frag", want: "/login/cb"},
		{name: "whitespace trimmed", redirectURL: "  https://example.com/login/cb  ", want: "/login/cb"},
		{name: "unparsable falls back", redirectURL: "://bad", want: oauth2CallbackPath},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &OAuth2Auth{RedirectURL: tc.redirectURL}
			if got := oauth2CallbackPathFor(cfg); got != tc.want {
				t.Fatalf("oauth2CallbackPathFor(%q) = %q, want %q", tc.redirectURL, got, tc.want)
			}
		})
	}
}

func TestIsOAuth2CallbackPath(t *testing.T) {
	custom := &OAuth2Auth{RedirectURL: "https://example.com/login/doreamon/callback"}

	if !isOAuth2CallbackPath("/login/doreamon/callback", custom) {
		t.Fatal("custom redirect_url path must be recognized as the callback")
	}
	if !isOAuth2CallbackPath(oauth2CallbackPath, custom) {
		t.Fatal("default callback path must stay accepted (in-flight logins across a redirect_url change)")
	}
	if isOAuth2CallbackPath("/", custom) {
		t.Fatal("a regular page must not be treated as the callback")
	}
	if isOAuth2CallbackPath("/login/doreamon/callback", &OAuth2Auth{}) {
		t.Fatal("custom path must not be treated as the callback without a matching redirect_url")
	}
	if !isOAuth2CallbackPath(oauth2CallbackPath, &OAuth2Auth{}) {
		t.Fatal("default callback path must be recognized when redirect_url is empty")
	}
}

// ---------------------------------------------------------------------------
// ValidateOAuth2 behaviour (no network: only the pre-token-exchange paths)
// ---------------------------------------------------------------------------

// serveOAuth2 runs one request through service.ValidateOAuth2 inside a zoox app.
func serveOAuth2(t *testing.T, svc *Service, method, target string, cookies ...*http.Cookie) (*httptest.ResponseRecorder, error) {
	t.Helper()

	app := zoox.New()
	app.Config.SecretKey = "test-secret"

	var handlerErr error
	app.Use(func(ctx *zoox.Context) {
		redirected, err := svc.ValidateOAuth2(ctx)
		if err != nil {
			handlerErr = err
			ctx.Status(http.StatusUnauthorized)
			return
		}
		if !redirected {
			ctx.String(http.StatusOK, "ok")
		}
	})

	req := httptest.NewRequest(method, target, nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	return rec, handlerErr
}

func doreamonTestService(redirectURL string) *Service {
	return &Service{
		Auth: Auth{
			Type: "oauth2",
			OAuth2: OAuth2Auth{
				Provider:     "doreamon",
				ClientID:     "c0602c25a2fafd47524f0f254856ff34",
				ClientSecret: "secret",
				RedirectURL:  redirectURL,
			},
		},
	}
}

// TestValidateOAuth2_CustomRedirectURLCallbackIsNotAProtectedPage reproduces the
// reported login loop: with a custom redirect_url whose path was not recognized as
// the callback, the callback request was treated as an unauthenticated page and
// redirected to /authorize again.
func TestValidateOAuth2_CustomRedirectURLCallbackIsNotAProtectedPage(t *testing.T) {
	svc := doreamonTestService("https://idp-office-dsh.ys.zcorky.com/login/doreamon/callback")

	rec, err := serveOAuth2(t, svc, http.MethodGet,
		"https://idp-office-dsh.ys.zcorky.com/login/doreamon/callback?code=abc123&state=deadbeefdeadbeefdeadbeefdeadbeef")
	if err != nil {
		t.Fatalf("callback must be handled, got error: %v", err)
	}
	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	location := rec.Header().Get("Location")
	if strings.Contains(location, "login.zcorky.com/authorize") {
		t.Fatalf("callback was treated as a protected page and bounced to /authorize (login loop): %s", location)
	}
	if location != "/?"+oauth2RetryQueryParam+"=1" {
		t.Fatalf("expected stale-state recovery redirect to the retry URL, got %q", location)
	}
}

// TestValidateOAuth2_ProtectedPageStillRedirectsToProvider keeps the normal flow intact.
func TestValidateOAuth2_ProtectedPageStillRedirectsToProvider(t *testing.T) {
	svc := doreamonTestService("https://idp-office-dsh.ys.zcorky.com/login/doreamon/callback")

	rec, err := serveOAuth2(t, svc, http.MethodGet, "https://idp-office-dsh.ys.zcorky.com/app?tab=1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "login.zcorky.com") || !strings.Contains(location, "authorize") {
		t.Fatalf("expected redirect to provider authorize URL, got %q", location)
	}
	if !strings.Contains(location, "redirect_uri=https%3A%2F%2Fidp-office-dsh.ys.zcorky.com%2Flogin%2Fdoreamon%2Fcallback") {
		t.Fatalf("expected configured redirect_uri in authorize URL, got %q", location)
	}
}

// TestValidateOAuth2_StaleStateSelfHealsOnce verifies a callback with an unknown
// CSRF state compacts the stale session and restarts the login (with a retry
// marker that does not depend on cookies) instead of returning a 401 that only an
// ingress restart could clear.
func TestValidateOAuth2_StaleStateSelfHealsOnce(t *testing.T) {
	svc := doreamonTestService("")

	rec, err := serveOAuth2(t, svc, http.MethodGet,
		"https://example.com/oauth2/callback?code=abc123&state=deadbeefdeadbeefdeadbeefdeadbeef")
	if err != nil {
		t.Fatalf("expected self-heal redirect, got error: %v", err)
	}
	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/?"+oauth2RetryQueryParam+"=1" {
		t.Fatalf("expected redirect to the bounded retry URL, got %q", loc)
	}
	if !hasRetryCookieForTest(rec) {
		t.Fatal("expected the short-lived retry marker cookie to be set")
	}
}

// TestValidateOAuth2_StaleStateCookieMarkerBoundsRetry verifies the recovery bound
// also works when the retry query parameter is lost in a redirect chain: the
// short-lived marker cookie alone forces a 401 on the next stale callback.
func TestValidateOAuth2_StaleStateCookieMarkerBoundsRetry(t *testing.T) {
	svc := doreamonTestService("")

	_, err := serveOAuth2(t, svc, http.MethodGet,
		"https://example.com/oauth2/callback?code=abc123&state=deadbeefdeadbeefdeadbeefdeadbeef",
		&http.Cookie{Name: oauth2RetryCookie, Value: "1"})
	if err == nil || !strings.Contains(err.Error(), "state mismatch") {
		t.Fatalf("expected a bounded state-mismatch error, got %v", err)
	}
}

func hasRetryCookieForTest(rec *httptest.ResponseRecorder) bool {
	for _, c := range rec.Result().Cookies() {
		if c.Name == oauth2RetryCookie && c.Value != "" {
			return true
		}
	}
	return false
}

// TestValidateOAuth2_StaleStateRetryIsBounded verifies the retry marker encoded in
// the CSRF state prevents an infinite /authorize <-> /callback loop, including for
// clients that drop every cookie (the marker travels in the URL/state, not a cookie).
func TestValidateOAuth2_StaleStateRetryIsBounded(t *testing.T) {
	svc := doreamonTestService("")

	rec, err := serveOAuth2(t, svc, http.MethodGet,
		"https://example.com/oauth2/callback?code=abc123&state="+url.QueryEscape(oauth2RetryStatePrefix+"deadbeefdeadbeefdeadbeefdeadbeef"))
	if err == nil {
		t.Fatalf("expected a state-mismatch error on the automatic retry, got redirect %q", rec.Header().Get("Location"))
	}
	if !strings.Contains(err.Error(), "state mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

// TestValidateOAuth2_RetryFlowMarksNextState verifies the retry URL parameter is
// turned into a marked CSRF state and does not leak into the post-login redirect.
func TestValidateOAuth2_RetryFlowMarksNextState(t *testing.T) {
	app := zoox.New()
	app.Config.SecretKey = "test-secret"
	svc := doreamonTestService("")

	var location string
	app.Use(func(ctx *zoox.Context) {
		redirected, err := svc.ValidateOAuth2(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !redirected {
			t.Fatal("expected a redirect to the provider")
		}
		location = ctx.Writer.Header().Get("Location")
	})

	req := httptest.NewRequest(http.MethodGet, "https://example.com/?"+oauth2RetryQueryParam+"=1", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	loginURL, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse provider URL: %v", err)
	}
	if !strings.HasPrefix(loginURL.Query().Get("state"), oauth2RetryStatePrefix) {
		t.Fatalf("expected a retry-marked state, got %q", loginURL.Query().Get("state"))
	}
}

// TestValidateOAuth2_ProviderErrorSurfaced verifies ?error=... callbacks report the
// provider error instead of a generic "missing code or state".
func TestValidateOAuth2_ProviderErrorSurfaced(t *testing.T) {
	svc := doreamonTestService("")

	_, err := serveOAuth2(t, svc, http.MethodGet,
		"https://example.com/oauth2/callback?error=access_denied&error_description=user+denied&state=deadbeef")
	if err == nil || !strings.Contains(err.Error(), "access_denied") || !strings.Contains(err.Error(), "user denied") {
		t.Fatalf("expected provider error to be surfaced, got %v", err)
	}
}

// TestValidateOAuth2_StaleCallbackWhenAlreadyLoggedIn verifies a replayed callback
// does not log an authenticated user out.
func TestValidateOAuth2_StaleCallbackWhenAlreadyLoggedIn(t *testing.T) {
	app := zoox.New()
	app.Config.SecretKey = "test-secret"
	svc := doreamonTestService("")

	var handlerErr error
	app.Use(func(ctx *zoox.Context) {
		ctx.Session().Set(sessOAuth2User, `{"id":"1","username":"u"}`)
		redirected, err := svc.ValidateOAuth2(ctx)
		handlerErr = err
		if !redirected && err == nil {
			ctx.String(http.StatusOK, "ok")
		}
	})

	req := httptest.NewRequest(http.MethodGet,
		"https://example.com/oauth2/callback?code=abc123&state=deadbeefdeadbeefdeadbeefdeadbeef", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if handlerErr != nil {
		t.Fatalf("unexpected error: %v", handlerErr)
	}
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/" {
		t.Fatalf("expected 302 to /, got %d %q", rec.Code, rec.Header().Get("Location"))
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == sessOAuth2User && c.Value == "" && c.MaxAge < 0 {
			t.Fatal("a replayed callback must not clear the authenticated session")
		}
	}
}

// ---------------------------------------------------------------------------
// session cookie budget
// ---------------------------------------------------------------------------

// TestOAuth2SessionCookieFitsBrowserLimit is the regression test for the
// "intermittent 401 on /oauth2/callback, only an ingress restart helps" bug:
// the OAuth2 session is a client-side cookie and browsers silently drop values
// over ~4096 bytes, after which the CSRF state can never be persisted again.
func TestOAuth2SessionCookieFitsBrowserLimit(t *testing.T) {
	app := zoox.New()
	app.Config.SecretKey = "test-secret"
	app.Use(func(ctx *zoox.Context) {
		// Worst case: a burst of unauthenticated requests each starting a flow with a
		// long post-login path (just under the per-URL cap, so it is stored as-is).
		longURL := "/api/v1/" + strings.Repeat("a", maxOAuth2FlowRedirectURLBytes-10)
		for i := 0; i < maxOAuth2Flows+8; i++ {
			saveOAuth2Flow(ctx, stateForIndex(i), longURL)
		}
		// Plus a large provider profile (permissions/groups are dropped).
		big := make([]string, 50)
		for i := range big {
			big[i] = strings.Repeat("p", 40)
		}
		ctx.Session().Set(sessOAuth2User, marshalOAuth2SessionUser(&gozooxoauth2.User{
			ID: "1", Username: "user", Email: strings.Repeat("e", 200) + "@example.com",
			Nickname: strings.Repeat("n", 200), Avatar: "https://example.com/" + strings.Repeat("a", 200),
			Permissions: big, Groups: big,
		}))
		ctx.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "https://example.com/app", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var value string
	for _, c := range rec.Result().Cookies() {
		if c.Name == "gsession" {
			value = c.Value
		}
	}
	if value == "" {
		t.Fatal("expected a gsession cookie to be written")
	}

	// HTTP cookies are limited to 4096 bytes including the name and "=".
	size := len("gsession=") + len(value)
	t.Logf("gsession cookie size with worst-case OAuth2 payload: %d bytes (browser limit 4096)", size)
	if size > 4096 {
		t.Fatalf("OAuth2 session cookie is %d bytes, browsers drop cookies over 4096 (state would be lost -> 401)", size)
	}
}

func TestOAuth2FlowMap_BoundedByBytes(t *testing.T) {
	app := zoox.New()
	app.Config.SecretKey = "test-secret"

	var count, size int
	var kept bool
	app.Use(func(ctx *zoox.Context) {
		// Path just under the per-URL cap, so flows are stored as-is and the JSON
		// byte cap (not the per-URL clamp) is what bounds the list.
		longURL := "/" + strings.Repeat("q", maxOAuth2FlowRedirectURLBytes-1)
		for i := 0; i < 40; i++ {
			saveOAuth2Flow(ctx, uniqueStateForTest(i), longURL)
		}
		flows := readOAuth2Flows(ctx)
		count = len(flows)
		size = oauth2FlowsSize(flows)
		// the most recent flow must survive pruning, and the oldest must be gone
		kept = flows[len(flows)-1].State == uniqueStateForTest(39)
		if flows[0].State == uniqueStateForTest(0) {
			t.Error("oldest flow should be evicted first")
		}
		ctx.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "https://example.com/app", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if count == 0 || count > maxOAuth2Flows {
		t.Fatalf("expected 1..%d flows, got %d", maxOAuth2Flows, count)
	}
	if size > maxOAuth2FlowsBytes {
		t.Fatalf("flow map is %d bytes, want <= %d", size, maxOAuth2FlowsBytes)
	}
	if count >= maxOAuth2Flows {
		t.Fatalf("expected the byte cap to prune below the count cap, got %d flows", count)
	}
	if !kept {
		t.Fatal("the newly created flow must survive pruning")
	}
}

func TestClampOAuth2RedirectURL(t *testing.T) {
	longPath := "/api/" + strings.Repeat("x", maxOAuth2FlowRedirectURLBytes)
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: "/"},
		{name: "short kept", in: "/app?tab=1", want: "/app?tab=1"},
		{name: "long drops query", in: "/app?q=" + strings.Repeat("q", 400), want: "/app"},
		{name: "too long path falls back", in: longPath, want: "/"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := clampOAuth2RedirectURL(tc.in); got != tc.want {
				t.Fatalf("clampOAuth2RedirectURL(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestOAuth2FlowLookupDoesNotConsume verifies a failed token exchange can be
// retried: looking a flow up must not delete it (only a completed login consumes it).
func TestOAuth2FlowLookupDoesNotConsume(t *testing.T) {
	app := zoox.New()
	app.Config.SecretKey = "test-secret"

	var peek1, peek2 bool
	var took bool
	var redirect string
	app.Use(func(ctx *zoox.Context) {
		saveOAuth2Flow(ctx, "state1", "/target")
		_, peek1 = lookupOAuth2Flow(ctx, "state1")
		_, peek2 = lookupOAuth2Flow(ctx, "state1")
		redirect, took = takeOAuth2Flow(ctx, "state1")
		if _, again := lookupOAuth2Flow(ctx, "state1"); again {
			t.Error("consumed flow must not be found again")
		}
		ctx.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "https://example.com/app", nil))
	if !peek1 || !peek2 {
		t.Fatal("lookupOAuth2Flow must not consume the flow")
	}
	if !took || redirect != "/target" {
		t.Fatalf("take: took=%v redirect=%q", took, redirect)
	}
}

func TestMarshalOAuth2SessionUser_Bounded(t *testing.T) {
	big := make([]string, 100)
	for i := range big {
		big[i] = strings.Repeat("p", 40)
	}
	raw := marshalOAuth2SessionUser(&gozooxoauth2.User{
		ID: "1", Username: "u", Email: "u@example.com",
		Nickname: strings.Repeat("n", 300), Avatar: strings.Repeat("a", 300),
		Permissions: big, Groups: big,
	})
	if len(raw) > maxOAuth2UserJSONBytes {
		t.Fatalf("user session payload is %d bytes, want <= %d", len(raw), maxOAuth2UserJSONBytes)
	}
	if strings.Contains(raw, "permissions") || strings.Contains(raw, "groups") {
		t.Fatalf("permissions/groups must not be stored in the cookie: %s", raw)
	}
	var u gozooxoauth2.User
	if err := json.Unmarshal([]byte(raw), &u); err != nil {
		t.Fatalf("session user must stay decodable by go-zoox/oauth2.User: %v", err)
	}
	if u.ID != "1" || u.Username != "u" {
		t.Fatalf("unexpected decoded user: %+v", u)
	}
}

// TestCompleteOAuth2Login verifies the successful-login path: the flow is
// consumed, only the trimmed user is stored (no token, no permissions), and the
// original URL is used as the redirect target.
func TestCompleteOAuth2Login(t *testing.T) {
	app := zoox.New()
	app.Config.SecretKey = "test-secret"

	var redirectURL, userJSON, token string
	var flowConsumed bool
	app.Use(func(ctx *zoox.Context) {
		saveOAuth2Flow(ctx, "state1", "/deep/link?tab=2")
		redirectURL = completeOAuth2Login(ctx, "state1", "/deep/link?tab=2", &gozooxoauth2.User{
			ID: "1", Username: "u", Email: "u@example.com",
			Permissions: []string{"a", "b"}, Groups: []string{"g"},
		})
		userJSON = ctx.Session().Get(sessOAuth2User)
		token = ctx.Session().Get(sessOAuth2Token)
		_, flowConsumed = lookupOAuth2Flow(ctx, "state1")
		ctx.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "https://example.com/app", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if redirectURL != "/deep/link?tab=2" {
		t.Fatalf("redirectURL = %q, want the original URL", redirectURL)
	}
	if flowConsumed {
		t.Fatal("successful login must consume the flow")
	}
	if token != "" {
		t.Fatalf("access token must not be stored in the session cookie, got %q", token)
	}
	if !strings.Contains(userJSON, `"username":"u"`) || strings.Contains(userJSON, "permissions") {
		t.Fatalf("unexpected stored user: %s", userJSON)
	}
}

// TestCompleteOAuth2Login_EmptyRedirectFallsBackToRoot covers flows saved without a URL.
func TestCompleteOAuth2Login_EmptyRedirectFallsBackToRoot(t *testing.T) {
	app := zoox.New()
	app.Config.SecretKey = "test-secret"
	var redirectURL string
	app.Use(func(ctx *zoox.Context) {
		redirectURL = completeOAuth2Login(ctx, "missing-state", "", nil)
		ctx.String(http.StatusOK, "ok")
	})
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "https://example.com/app", nil))
	if redirectURL != "/" {
		t.Fatalf("redirectURL = %q, want /", redirectURL)
	}
}

func uniqueStateForTest(i int) string {
	return fmt.Sprintf("%032x", i)
}

// lastSessionCookieSize returns name+value bytes of the latest gsession Set-Cookie.
func lastSessionCookieSize(rec *httptest.ResponseRecorder) int {
	var value string
	for _, c := range rec.Result().Cookies() {
		if c.Name == "gsession" {
			value = c.Value
		}
	}
	if value == "" {
		return 0
	}
	return len("gsession=") + len(value)
}

// TestOAuth2LegacyOversizedCookieIsRepaired reproduces the reported production
// state: an old-version session cookie (64 flows + access token + full profile)
// grows past the ~4KB browser limit, so the browser drops every update and the
// callback can never persist its CSRF state. Recovery must answer with one small
// cookie header (not a re-flushed oversized payload) so the browser accepts it and
// the login works again without restarting ingress.
func TestOAuth2LegacyOversizedCookieIsRepaired(t *testing.T) {
	app := zoox.New()
	app.Config.SecretKey = "test-secret"
	svc := doreamonTestService("")

	var handlerErr error
	app.Use(func(ctx *zoox.Context) {
		if ctx.Query().Get("seed").String() == "1" {
			legacyFlows := map[string]string{}
			for i := 0; i < 64; i++ {
				legacyFlows[uniqueStateForTest(i)] = "/api/v1/devices?page=1&size=50&filter=" + strings.Repeat("a", 250)
			}
			legacyJSON, _ := json.Marshal(legacyFlows)
			ctx.Session().Set(sessOAuth2Flows, string(legacyJSON))
			ctx.Session().Set(sessOAuth2Token, `{"access_token":"`+strings.Repeat("t", 400)+`","refresh_token":"ref"}`)
			ctx.Session().Set(sessOAuth2State, "legacy-state")
			ctx.Session().Set(sessOAuth2Redirect, "/legacy/path")
			ctx.String(http.StatusOK, "seeded")
			return
		}
		redirected, err := svc.ValidateOAuth2(ctx)
		handlerErr = err
		if !redirected && err == nil {
			ctx.String(http.StatusOK, "ok")
		}
	})

	// 1. Seed an old-version session and confirm it is over the browser limit.
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "https://example.com/app?seed=1", nil))
	before := lastSessionCookieSize(rec)
	if before <= 4096 {
		t.Fatalf("expected the legacy OAuth2 session cookie to exceed the browser limit, got %d bytes", before)
	}
	seedCookie := findCookieForTest(rec, "gsession")
	if seedCookie == nil {
		t.Fatal("expected the legacy session cookie")
	}

	// 2. A stale callback must answer with only small cookie headers (deletion),
	// never by re-flushing the oversized payload.
	req := httptest.NewRequest(http.MethodGet,
		"https://example.com/oauth2/callback?code=abc123&state=deadbeefdeadbeefdeadbeefdeadbeef", nil)
	req.AddCookie(&http.Cookie{Name: seedCookie.Name, Value: seedCookie.Value})
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if handlerErr != nil {
		t.Fatalf("expected self-heal redirect, got error: %v", handlerErr)
	}
	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/?"+oauth2RetryQueryParam+"=1" {
		t.Fatalf("expected redirect to the retry URL, got %q", loc)
	}
	if max := maxSetCookieSize(rec); max > 4096 {
		t.Fatalf("recovery emitted a %d-byte Set-Cookie header (browsers and proxies reject it)", max)
	}
	deleted := false
	for _, c := range rec.Result().Cookies() {
		if c.Name == "gsession" && c.MaxAge < 0 {
			deleted = true
		}
	}
	if !deleted {
		t.Fatal("expected the stale gateway session cookie to be deleted")
	}
}

// maxSetCookieSize returns the largest name+value size across Set-Cookie headers.
func maxSetCookieSize(rec *httptest.ResponseRecorder) int {
	max := 0
	for _, header := range rec.Header().Values("Set-Cookie") {
		value := header
		if idx := strings.Index(header, ";"); idx >= 0 {
			value = header[:idx]
		}
		if len(value) > max {
			max = len(value)
		}
	}
	return max
}

// TestOAuth2FlowsLegacyMapMigration verifies the JSON-map format written by older
// versions is still read, then rewritten in the ordered array format.
func TestOAuth2FlowsLegacyMapMigration(t *testing.T) {
	app := zoox.New()
	app.Config.SecretKey = "test-secret"

	var count int
	var rewritten string
	app.Use(func(ctx *zoox.Context) {
		ctx.Session().Set(sessOAuth2Flows, `{"aaaa":"/a","bbbb":"/b"}`)
		count = len(readOAuth2Flows(ctx))
		saveOAuth2Flow(ctx, "cccc", "/c")
		rewritten = ctx.Session().Get(sessOAuth2Flows)
		ctx.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "https://example.com/app", nil))
	if count != 2 {
		t.Fatalf("legacy flows: got %d, want 2", count)
	}
	if !strings.HasPrefix(rewritten, "[[") {
		t.Fatalf("expected the ordered array format after a save, got %q", rewritten)
	}
}

func findCookieForTest(rec *httptest.ResponseRecorder, name string) *http.Cookie {
	var found *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == name {
			found = c
		}
	}
	return found
}
