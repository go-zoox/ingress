package service

import (
	"testing"

	"github.com/go-zoox/ingress/core/admin/model"
)

func TestOverviewBuilder_SnapshotDefaults(t *testing.T) {
	b := NewOverviewBuilder(nil, nil, nil, nil, nil, nil, nil, nil)
	snap := b.Snapshot("")
	if snap.Window != "15m" {
		t.Fatalf("window = %q, want 15m", snap.Window)
	}
	if snap.Certs == nil {
		t.Fatal("expected non-nil certs slice")
	}
	if snap.WAFBlocks == nil {
		t.Fatal("expected non-nil waf_blocks slice")
	}
	if snap.ParseIssues == nil {
		t.Fatal("expected non-nil parse_issues slice")
	}
	if snap.Revisions == nil {
		t.Fatal("expected non-nil revisions slice")
	}
}

func TestOverviewWAFBlocks(t *testing.T) {
	rows := []model.WAFEvent{
		{ID: 1, Action: "audit"},
		{ID: 2, Action: "block"},
		{ID: 3, Action: "block"},
		{ID: 4, Action: "block"},
	}
	out := overviewWAFBlocks(rows, 2)
	if len(out) != 2 {
		t.Fatalf("len = %d, want 2", len(out))
	}
	if out[0].ID != 2 || out[1].ID != 3 {
		t.Fatalf("unexpected ids: %+v", out)
	}
}
