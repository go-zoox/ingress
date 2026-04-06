package service

import "testing"

func TestCheckDNS(t *testing.T) {
	// RFC 2606 / RFC 6761: names under .invalid must not resolve in the public DNS.
	s1 := &Service{
		Name: "nonexistent.invalid",
	}
	_, err := s1.CheckDNS()
	if err == nil {
		t.Fatal("expected DNS lookup to fail for nonexistent.invalid")
	}

	s2 := &Service{
		Name: "example.com",
	}
	_, err = s2.CheckDNS()
	if err != nil {
		t.Fatal(err)
	}
}
