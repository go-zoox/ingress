package geoip

import "testing"

func TestLookupFallback_DemoIP(t *testing.T) {
	p, ok := lookupFallback("203.0.113.44")
	if !ok || p.Label != "北京" {
		t.Fatalf("lookup=%+v ok=%v", p, ok)
	}
}

func TestLookupFallback_Private(t *testing.T) {
	_, ok := lookupFallback("10.0.0.5")
	if ok {
		t.Fatal("expected private ip skipped")
	}
}

func TestLookupFallback_PublicApprox(t *testing.T) {
	p, ok := lookupFallback("8.8.8.8")
	if !ok || !p.Approx {
		t.Fatalf("lookup=%+v ok=%v", p, ok)
	}
}

func TestDefaultIngress(t *testing.T) {
	ing := defaultIngress(Config{})
	if ing.Lat != 31.2304 || ing.Label != "Ingress" {
		t.Fatalf("ingress=%+v", ing)
	}
	ing = defaultIngress(Config{IngressLat: 1, IngressLng: 2, IngressLabel: "Edge"})
	if ing.Label != "Edge" {
		t.Fatalf("ingress=%+v", ing)
	}
}

func TestReconfigure(t *testing.T) {
	s1, err := Init(Config{IngressLabel: "A"})
	if err != nil {
		t.Fatal(err)
	}
	s2, err := Reconfigure(Config{IngressLabel: "B"})
	if err != nil {
		t.Fatal(err)
	}
	if s2.Ingress().Label != "B" {
		t.Fatalf("ingress=%+v", s2.Ingress())
	}
	if GlobalIngress().Label != "B" {
		t.Fatalf("global ingress=%+v", GlobalIngress())
	}
	_ = s1.Close()
	_ = s2.Close()
}

func TestPickName(t *testing.T) {
	if pickName(map[string]string{"zh-CN": "上海", "en": "Shanghai"}) != "上海" {
		t.Fatal("expected zh-CN")
	}
	if pickName(map[string]string{"en": "Tokyo"}) != "Tokyo" {
		t.Fatal("expected en")
	}
}
