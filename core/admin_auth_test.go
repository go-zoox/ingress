package core

import "testing"

func TestEffectiveAdminAuthType(t *testing.T) {
	if EffectiveAdminAuthType("") != "none" {
		t.Fatal("empty auth type should default to none")
	}
	if EffectiveAdminAuthType("basic") != "basic" {
		t.Fatal("expected basic")
	}
	if EffectiveAdminAuthType("oauth") != "oauth" {
		t.Fatal("expected oauth")
	}
	if EffectiveAdminAuthType("NONE") != "none" {
		t.Fatal("expected normalized none")
	}
	if EffectiveAdminAuthType("unknown") != "none" {
		t.Fatal("unknown auth type should default to none")
	}
}

func TestAdminAuthValidateDefaultNone(t *testing.T) {
	if err := (AdminAuth{}).Validate(); err != nil {
		t.Fatalf("default admin auth should validate: %v", err)
	}
}
