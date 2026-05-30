package config

import "testing"

func TestEffectiveAuthType(t *testing.T) {
	if EffectiveAuthType("") != "none" {
		t.Fatal("empty auth type should default to none")
	}
	if EffectiveAuthType("basic") != "basic" {
		t.Fatal("expected basic")
	}
	if EffectiveAuthType("oauth") != "oauth" {
		t.Fatal("expected oauth")
	}
	if EffectiveAuthType("unknown") != "none" {
		t.Fatal("unknown auth type should default to none")
	}
}

func TestAuthValidateDefaultNone(t *testing.T) {
	if err := (&Auth{}).Validate(); err != nil {
		t.Fatalf("default auth should validate: %v", err)
	}
}
