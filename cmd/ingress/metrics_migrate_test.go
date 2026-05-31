package main

import (
	"testing"
	"time"
)

func TestParseMigrateDuration(t *testing.T) {
	d, ok := parseMigrateDuration("7d")
	if !ok || d != 7*24*time.Hour {
		t.Fatalf("7d: ok=%v d=%v", ok, d)
	}
	d, ok = parseMigrateDuration("24h")
	if !ok || d != 24*time.Hour {
		t.Fatalf("24h: ok=%v d=%v", ok, d)
	}
	if _, ok := parseMigrateDuration("not-a-duration"); ok {
		t.Fatal("expected invalid duration")
	}
}

func TestParseMigrateTime_since7d(t *testing.T) {
	before := time.Now()
	tm, err := parseMigrateTime("7d", true)
	if err != nil {
		t.Fatal(err)
	}
	after := time.Now()
	wantMin := before.Add(-7 * 24 * time.Hour)
	wantMax := after.Add(-7 * 24 * time.Hour)
	if tm.Before(wantMin) || tm.After(wantMax) {
		t.Fatalf("time=%v want between %v and %v", tm, wantMin, wantMax)
	}
}
