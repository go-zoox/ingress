package service

import "testing"

func TestRewrite(t *testing.T) {
	s := &Service{
		Request: Request{
			Path: RequestPath{
				Rewrites: []string{
					"/api:/",
				},
			},
		},
	}

	r := s.Rewrite()
	if len(r) != 1 {
		t.Fatalf("expected 1 rewrite, got %d", len(r))
	}
	if r[0].From != "/api" {
		t.Fatalf("expected /api, got %s", r[0].From)
	}
	if r[0].To != "/" {
		t.Fatalf("expected /, got %s", r[0].To)
	}
}

func TestRewriteWithRegExp(t *testing.T) {
	s := &Service{
		Request: Request{
			Path: RequestPath{
				Rewrites: []string{
					"^/api/(.*):/$1",
				},
			},
		},
	}

	r := s.Rewrite()
	if len(r) != 1 {
		t.Fatalf("expected 1 rewrite, got %d", len(r))
	}

	if r[0].From != "^/api/(.*)" {
		t.Fatalf("expected /api/(.*), got %s", r[0].From)
	}

	if r[0].To != "/$1" {
		t.Fatalf("expected /$1, got %s", r[0].To)
	}
}

func TestRewriteInvalid(t *testing.T) {
	s := &Service{
		Request: Request{
			Path: RequestPath{
				Rewrites: []string{
					"/api",
					"/api:/api2:/api3",
					"",
				},
			},
		},
	}

	r := s.Rewrite()
	if len(r) != 0 {
		t.Fatalf("expected 0 rewrite, got %d", len(r))
	}
}
