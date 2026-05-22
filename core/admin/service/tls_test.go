package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWildcardMatch(t *testing.T) {
	cases := []struct {
		pattern string
		host    string
		want    bool
	}{
		{"*.example.com", "api.example.com", true},
		{"*.example.com", "example.com", false},
		{"api.example.com", "api.example.com", true},
		{"api.example.com", "cdn.example.com", false},
	}
	for _, tc := range cases {
		if got := wildcardMatch(tc.pattern, tc.host); got != tc.want {
			t.Fatalf("wildcardMatch(%q, %q)=%v want %v", tc.pattern, tc.host, got, tc.want)
		}
	}
}

func TestParseSampleCert(t *testing.T) {
	root := filepath.Join("..", "..", "..", "examples", "admin-console", "certs", "api.example.com.pem")
	if _, err := os.Stat(root); err != nil {
		t.Skip("sample cert not found")
	}
	data, err := os.ReadFile(root)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := parseX509CertPEM(data)
	if err != nil {
		t.Fatal(err)
	}
	if !certMatchesDomain(cert, "api.example.com") {
		t.Fatal("expected domain match")
	}
}
