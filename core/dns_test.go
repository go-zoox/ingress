package core

import "testing"

func TestCheckDNS(t *testing.T) {
	s1 := &core{}
	ok, _, err := s1.CheckDNS("notfound.domain")
	if err == nil {
		t.Fatal("should be error")
	}
	if ok {
		t.Fatal("should be false")
	}

	s2 := &core{}
	ok, _, err = s2.CheckDNS("baidu.com")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("should be true")
	}
}
