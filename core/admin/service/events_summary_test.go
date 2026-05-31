package service

import (
	"testing"

	"github.com/go-zoox/gormx"
	"github.com/go-zoox/ingress/core/admin/model"
)

func TestBuildEventsTabSummary_openIncludesHealthAndTLS(t *testing.T) {
	s := BuildEventsTabSummary("open", 120, 5, 2, 3)
	if s.Total != 130 {
		t.Fatalf("total=%d want 130", s.Total)
	}
}

func TestBuildEventsTabSummary_resolvedOnlyPersisted(t *testing.T) {
	s := BuildEventsTabSummary("resolved", 10, 4, 99, 99)
	if s.Total != 14 {
		t.Fatalf("total=%d want 14", s.Total)
	}
	if s.HealthDown != 99 {
		t.Fatalf("health_down should be preserved in struct but not in total")
	}
}

func TestCountBlockWAFEvents(t *testing.T) {
	setupEventsBulkDB(t)
	db := gormx.GetDB()
	rows := []model.WAFEvent{
		{Action: "block", Rule: "a", Status: "open"},
		{Action: "block", Rule: "b", Status: "resolved"},
		{Action: "audit", Rule: "c", Status: "open"},
	}
	for i := range rows {
		if err := db.Create(&rows[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	audit := &Audit{}
	n, err := audit.CountBlockWAFEvents("open")
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("open block count=%d want 1", n)
	}
}
