package core

import "testing"

func TestCheckDNS(t *testing.T) {
	s1 := &core{}
	_, err := s1.CheckDNS("notfound.domain")
	if err == nil {
		t.Fatal("should be error")
	}

	s2 := &core{}
	_, err = s2.CheckDNS("baidu.com")
	if err != nil {
		t.Fatal(err)
	}
}
