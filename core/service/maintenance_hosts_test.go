package service

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestMaintenanceHostList_UnmarshalYAML_StringAndObject(t *testing.T) {
	var list MaintenanceHostList
	err := yaml.Unmarshal([]byte(`
- app.example.com
- host: legacy.example.com
  window:
    start: "2026-05-30T02:00:00+08:00"
    end: "2026-05-30T06:00:00+08:00"
`), &list)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("len=%d", len(list))
	}
	if list[0].Host != "app.example.com" || list[0].Window.Configured() {
		t.Fatalf("entry0=%+v", list[0])
	}
	if list[1].Host != "legacy.example.com" || !list[1].Window.Configured() {
		t.Fatalf("entry1=%+v", list[1])
	}
}

func TestMaintenanceHostList_Patterns(t *testing.T) {
	list := MaintenanceHostList{
		{Host: "a.example.com"},
		{Host: "  "},
		{Host: "b.example.com"},
	}
	got := list.Patterns()
	if len(got) != 2 || got[0] != "a.example.com" || got[1] != "b.example.com" {
		t.Fatalf("patterns=%v", got)
	}
}
