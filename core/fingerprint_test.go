package core

import "testing"

func TestContentHash_stable(t *testing.T) {
	a := ContentHash("version: v1\nport: 8080\n")
	b := ContentHash("version: v1\nport: 8080\n")
	if a != b || len(a) != 16 {
		t.Fatalf("hash=%q", a)
	}
}
