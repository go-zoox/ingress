package core

import (
	"strings"
	"testing"
)

func TestSetScenariosActiveYAML_missingBlock(t *testing.T) {
	_, err := SetScenariosActiveYAML("port: 8080\n", "live")
	if err == nil {
		t.Fatal("expected error when scenarios block missing")
	}
}

func TestSetScenariosActiveYAML_default(t *testing.T) {
	in := "scenarios:\n  active: live\n  items:\n    - id: live\n"
	out, err := SetScenariosActiveYAML(in, DefaultScenarioID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "active: default") {
		t.Fatalf("expected active default, got:\n%s", out)
	}
}

func TestSetScenariosActiveYAML_unknownID(t *testing.T) {
	in := "scenarios:\n  active: daily\n  items:\n    - id: daily\n"
	_, err := SetScenariosActiveYAML(in, "live")
	if err == nil {
		t.Fatal("expected unknown id error")
	}
}
