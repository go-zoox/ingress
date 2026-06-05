package core

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-zoox/ingress/core/rule"
)

func TestEffectiveRedirectBehavior(t *testing.T) {
	tests := []struct {
		name                    string
		duration                string
		preserveRequest         bool
		permanent               bool
		withOriginMethodAndBody  bool
		wantPermanent           bool
		wantPreserve            bool
		wantStatus              int
		wantErr                 bool
	}{
		{
			name:       "default temporary safe",
			wantStatus: http.StatusFound,
		},
		{
			name:          "duration permanent",
			duration:      redirectDurationPermanent,
			wantPermanent: true,
			wantStatus:    http.StatusMovedPermanently,
		},
		{
			name:            "preserve temporary",
			preserveRequest: true,
			wantPreserve:    true,
			wantStatus:      http.StatusTemporaryRedirect,
		},
		{
			name:            "preserve permanent",
			duration:        redirectDurationPermanent,
			preserveRequest: true,
			wantPermanent:   true,
			wantPreserve:    true,
			wantStatus:      http.StatusPermanentRedirect,
		},
		{
			name:          "legacy permanent",
			permanent:     true,
			wantPermanent: true,
			wantStatus:    http.StatusMovedPermanently,
		},
		{
			name:                    "legacy with origin",
			withOriginMethodAndBody: true,
			wantPreserve:            true,
			wantStatus:              http.StatusTemporaryRedirect,
		},
		{
			name:                    "legacy permanent strict",
			permanent:               true,
			withOriginMethodAndBody: true,
			wantPermanent:           true,
			wantPreserve:            true,
			wantStatus:              http.StatusPermanentRedirect,
		},
		{
			name:     "invalid duration",
			duration: "forever",
			wantErr:  true,
		},
		{
			name:      "duration temporary conflicts permanent",
			duration:  redirectDurationTemporary,
			permanent: true,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := effectiveRedirectBehavior(tt.duration, tt.preserveRequest, tt.permanent, tt.withOriginMethodAndBody)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Permanent != tt.wantPermanent {
				t.Fatalf("Permanent = %v want %v", got.Permanent, tt.wantPermanent)
			}
			if got.PreserveRequest != tt.wantPreserve {
				t.Fatalf("PreserveRequest = %v want %v", got.PreserveRequest, tt.wantPreserve)
			}
			if code := redirectStatusCode(got); code != tt.wantStatus {
				t.Fatalf("status = %d want %d", code, tt.wantStatus)
			}
		})
	}
}

func TestValidateRuleRedirect(t *testing.T) {
	if err := validateRuleRedirect(rule.Redirect{URL: "https://x.example/"}, "rules[0].backend.redirect"); err != nil {
		t.Fatal(err)
	}
	if err := validateRuleRedirect(rule.Redirect{
		URL:       "https://x.example/",
		Duration:  redirectDurationTemporary,
		Permanent: true,
	}, "rules[0].backend.redirect"); err == nil {
		t.Fatal("expected conflict error")
	}
}

func TestValidateRedirectFromHTTP(t *testing.T) {
	if err := validateRedirectFromHTTP(RedirectFromHTTP{Enabled: true, Duration: "bad"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestBuild_BackendRedirect_DurationPreserveRequest(t *testing.T) {
	tests := []struct {
		name     string
		redirect rule.Redirect
		wantCode int
	}{
		{
			name: "scheme a temporary safe",
			redirect: rule.Redirect{
				URL:      "https://target.example/over",
				Duration: redirectDurationTemporary,
			},
			wantCode: http.StatusFound,
		},
		{
			name: "scheme a permanent safe",
			redirect: rule.Redirect{
				URL:      "https://target.example/over",
				Duration: redirectDurationPermanent,
			},
			wantCode: http.StatusMovedPermanently,
		},
		{
			name: "scheme a temporary strict",
			redirect: rule.Redirect{
				URL:             "https://target.example/over",
				Duration:        redirectDurationTemporary,
				PreserveRequest: true,
			},
			wantCode: http.StatusTemporaryRedirect,
		},
		{
			name: "scheme a permanent strict",
			redirect: rule.Redirect{
				URL:             "https://target.example/over",
				Duration:        redirectDurationPermanent,
				PreserveRequest: true,
			},
			wantCode: http.StatusPermanentRedirect,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Port: 8080,
				Rules: []rule.Rule{
					{
						Host: "redirect-status.example.com",
						Backend: rule.Backend{
							Redirect: tt.redirect,
						},
					},
				},
			}

			c, err := New("test-version", cfg)
			if err != nil {
				t.Fatalf("New: %v", err)
			}

			ins := c.(*core)
			if err := ins.build(); err != nil {
				t.Fatalf("build: %v", err)
			}

			req := httptest.NewRequest(http.MethodGet, "http://redirect-status.example.com/path", nil)
			rec := httptest.NewRecorder()
			ins.app.ServeHTTP(rec, req)

			if rec.Code != tt.wantCode {
				t.Fatalf("expected status %d, got %d", tt.wantCode, rec.Code)
			}
		})
	}
}
