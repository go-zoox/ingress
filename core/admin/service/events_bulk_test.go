package service

import (
	"path/filepath"
	"testing"

	"github.com/go-zoox/gormx"
	"github.com/go-zoox/ingress/core/admin/model"
)

func setupEventsBulkDB(t *testing.T) {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "events-bulk.db")
	if err := gormx.LoadDB("sqlite", dsn); err != nil {
		t.Fatal(err)
	}
	if err := gormx.GetDB().AutoMigrate(model.MigrateModels()...); err != nil {
		t.Fatal(err)
	}
}

func TestSetAllOpenBlockWAFEventStatus(t *testing.T) {
	setupEventsBulkDB(t)
	db := gormx.GetDB()
	rows := []model.WAFEvent{
		{Action: "block", Rule: "a", Host: "h1", Path: "/1", Status: "open"},
		{Action: "block", Rule: "b", Host: "h2", Path: "/2", Status: ""},
		{Action: "block", Rule: "c", Host: "h3", Path: "/3", Status: "resolved"},
		{Action: "audit", Rule: "d", Host: "h4", Path: "/4", Status: "open"},
	}
	for i := range rows {
		if err := db.Create(&rows[i]).Error; err != nil {
			t.Fatal(err)
		}
	}

	audit := &Audit{}
	n, err := audit.SetAllOpenBlockWAFEventStatus("resolved", "bulk")
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("updated=%d want 2", n)
	}

	var openCount int64
	db.Model(&model.WAFEvent{}).
		Where("action = ?", "block").
		Where("(status = ? OR status = '' OR status IS NULL)", "open").
		Count(&openCount)
	if openCount != 0 {
		t.Fatalf("open block count=%d want 0", openCount)
	}
}

func TestParseIssuesSetAllOpenStatus(t *testing.T) {
	setupEventsBulkDB(t)
	db := gormx.GetDB()
	issues := []model.AccessLogParseIssue{
		{Fingerprint: "a", Reason: "missing_host", Status: "open", HitCount: 1},
		{Fingerprint: "b", Reason: "missing_host", Status: "open", HitCount: 2},
		{Fingerprint: "c", Reason: "missing_host", Status: "ignored", HitCount: 3},
	}
	for i := range issues {
		if err := db.Create(&issues[i]).Error; err != nil {
			t.Fatal(err)
		}
	}

	p := NewParseIssues()
	n, err := p.SetAllOpenStatus("ignored", "all")
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("updated=%d want 2", n)
	}

	var openCount int64
	db.Model(&model.AccessLogParseIssue{}).Where("status = ?", "open").Count(&openCount)
	if openCount != 0 {
		t.Fatalf("open parse issues=%d want 0", openCount)
	}
}
