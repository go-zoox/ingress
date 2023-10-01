package service

import "testing"

func TestCheckDNS(t *testing.T) {
	s1 := &Service{
		Name: "notfound.domain",
	}
	_, err := s1.CheckDNS()
	if err == nil {
		t.Fatal("should be error")
	}

	s2 := &Service{
		Name: "baidu.com",
	}
	_, err = s2.CheckDNS()
	if err != nil {
		t.Fatal(err)
	}
}
