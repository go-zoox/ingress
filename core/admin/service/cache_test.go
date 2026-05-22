package service

import "testing"

func TestTopPathHitRates(t *testing.T) {
	total := map[string]int{
		"/assets/app.js": 100,
		"/api/users":     50,
		"/search":        30,
	}
	hits := map[string]int{
		"/assets/app.js": 72,
		"/api/users":     5,
		"/search":        0,
	}
	got := topPathHitRates(total, hits, 2)
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].Path != "/assets/app.js" || got[0].Hits != 72 || got[0].Total != 100 {
		t.Fatalf("first: %+v", got[0])
	}
	if got[1].Path != "/api/users" {
		t.Fatalf("second path: %s", got[1].Path)
	}
}
