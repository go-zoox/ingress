package service

import (
	"testing"

	"github.com/go-zoox/ingress/core/admin/model"
	"github.com/go-zoox/ingress/core/admin/service/geoip"
)

func TestBuildWAFVisualization_Fallback(t *testing.T) {
	events := []model.WAFEvent{
		{Action: "block", ClientIP: "203.0.113.44"},
		{Action: "audit", ClientIP: "203.0.113.44"},
		{Action: "block", ClientIP: "10.0.0.5"},
	}
	viz := BuildWAFVisualization(events)
	if viz.Total != 3 {
		t.Fatalf("total=%d", viz.Total)
	}
	if viz.UnknownIPs != 1 {
		t.Fatalf("unknown=%d", viz.UnknownIPs)
	}
	if len(viz.Points) != 1 || viz.Points[0].Count != 2 {
		t.Fatalf("points=%+v", viz.Points)
	}
	if viz.Points[0].Label != "北京" {
		t.Fatalf("label=%q", viz.Points[0].Label)
	}
}

func TestGeoIPLookup_DemoIP(t *testing.T) {
	p, ok := geoip.Lookup("203.0.113.44")
	if !ok || p.Label != "北京" || p.Approx {
		t.Fatalf("lookup=%+v ok=%v", p, ok)
	}
}

func TestGeoIPLookup_Private(t *testing.T) {
	_, ok := geoip.Lookup("10.0.0.5")
	if ok {
		t.Fatal("expected private ip to be skipped")
	}
}
