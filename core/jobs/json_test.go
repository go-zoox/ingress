package jobs

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestJobParams_JSONUsesSnakeCaseKeys(t *testing.T) {
	p := JobParams{
		Script: "echo hi",
		Engine: ScriptEngineShell,
		Shell:  "sh",
		URL:    "http://127.0.0.1/health",
		Method: "GET",
	}
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	for _, key := range []string{`"script"`, `"engine"`, `"shell"`, `"url"`, `"method"`} {
		if !strings.Contains(s, key) {
			t.Fatalf("json %s missing %s", s, key)
		}
	}
	if strings.Contains(s, `"Script"`) {
		t.Fatalf("json should not expose PascalCase Script: %s", s)
	}

	var decoded JobParams
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Script != p.Script {
		t.Fatalf("round-trip script = %q", decoded.Script)
	}
}

func TestItem_JSONUnmarshalScriptParam(t *testing.T) {
	raw := []byte(`{"id":"x","name":"n","kind":"script","schedule":"* * * * *","enabled":true,"params":{"script":"echo ok","engine":"shell","shell":"sh"}}`)
	var item Item
	if err := json.Unmarshal(raw, &item); err != nil {
		t.Fatal(err)
	}
	if item.Params.Script != "echo ok" {
		t.Fatalf("script = %q", item.Params.Script)
	}
}
