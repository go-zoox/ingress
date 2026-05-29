package service

import "testing"

func TestHostTrafficStats_pvUv(t *testing.T) {
	entries := []AccessEntry{
		{Host: "api.example.com", ClientIP: "203.0.113.1", RealIP: "203.0.113.1"},
		{Host: "api.example.com", ClientIP: "203.0.113.1", RealIP: "203.0.113.1"},
		{Host: "api.example.com", ClientIP: "10.0.0.1", RealIP: "203.0.113.2"},
		{Host: "cdn.example.com", ClientIP: "198.51.100.8", RealIP: "198.51.100.8"},
		{Host: "cdn.example.com", ClientIP: "198.51.100.9", RealIP: "-"},
	}
	stats := hostTrafficStats(entries, 10)
	if len(stats) != 2 {
		t.Fatalf("len=%d want 2", len(stats))
	}
	if stats[0].Name != "api.example.com" || stats[0].PV != 3 || stats[0].UV != 2 {
		t.Fatalf("api: %+v", stats[0])
	}
	if stats[1].Name != "cdn.example.com" || stats[1].PV != 2 || stats[1].UV != 2 {
		t.Fatalf("cdn: %+v", stats[1])
	}
}

func TestVisitorIP_prefersRealIP(t *testing.T) {
	e := AccessEntry{ClientIP: "10.0.0.5", RealIP: "203.0.113.44"}
	if got := visitorIP(e); got != "203.0.113.44" {
		t.Fatalf("got %q", got)
	}
	e = AccessEntry{ClientIP: "10.0.0.5", RealIP: "-"}
	if got := visitorIP(e); got != "10.0.0.5" {
		t.Fatalf("fallback got %q", got)
	}
	e = AccessEntry{ClientIP: "127.0.0.1:53272"}
	if got := visitorIP(e); got != "127.0.0.1" {
		t.Fatalf("strip port got %q", got)
	}
}

func TestParseAccessLine_realIP(t *testing.T) {
	line := `203.0.113.44 api.example.com -> api.internal:8080 "GET / HTTP/1.1" 200 12ms cache_hit=0 waf_block=0 real_ip=203.0.113.44 referer=-`
	e, ok := parseAccessLine(line)
	if !ok {
		t.Fatal("parse failed")
	}
	if e.RealIP != "203.0.113.44" {
		t.Fatalf("real_ip=%q", e.RealIP)
	}
	if visitorIP(e) != "203.0.113.44" {
		t.Fatalf("visitor=%q", visitorIP(e))
	}
}
