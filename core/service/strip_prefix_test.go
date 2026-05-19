package service

import "testing"

func TestRewrite_stripPrefixExpanded(t *testing.T) {
	s := &Service{
		Request: Request{
			Path: RequestPath{
				Rewrites: []string{
					"^/api/dashboard/?(.*):/$1",
				},
			},
		},
	}

	cases := []struct {
		in, want string
	}{
		{"/api/dashboard", "/"},
		{"/api/dashboard/", "/"},
		{"/api/dashboard/foo", "/foo"},
		{"/api/dashboard/foo/bar", "/foo/bar"},
	}
	for _, tc := range cases {
		got := s.Rewrite(tc.in)
		if got != tc.want {
			t.Errorf("Rewrite(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
