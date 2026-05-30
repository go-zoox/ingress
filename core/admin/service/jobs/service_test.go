package jobs

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	ingjobs "github.com/go-zoox/ingress/core/jobs"
)

func TestLoadAndMarshalJobsModule(t *testing.T) {
	content := testIngressBase + `
jobs:
  builtins:
    purge_waf_events:
      enabled: true
      schedule: "0 3 * * *"
  items:
    - id: nightly
      name: Nightly
      kind: http_call
      schedule: "0 1 * * *"
      enabled: true
      params:
        url: http://127.0.0.1/health
`
	jcfg, err := loadJobsFromContent(content)
	if err != nil {
		t.Fatal(err)
	}
	if len(jcfg.Items) != 1 || jcfg.Items[0].ID != "nightly" {
		t.Fatalf("items = %+v", jcfg.Items)
	}
	if jcfg.Builtins["purge_waf_events"].Schedule != "0 3 * * *" {
		t.Fatalf("builtins = %+v", jcfg.Builtins)
	}

	modYAML, err := marshalJobsModule(jcfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(modYAML, "nightly") {
		t.Fatalf("yaml = %s", modYAML)
	}

	round, err := loadJobsFromContent(testIngressBase + "\n" + modYAML)
	if err != nil {
		t.Fatal(err)
	}
	if len(round.Items) != 1 || round.Items[0].Name != "Nightly" {
		t.Fatalf("round = %+v", round.Items)
	}
}

func TestService_CreateUpdateDeleteItem(t *testing.T) {
	svc := newTestService(t, testIngressBase)

	item := ingjobs.Item{
		ID:       "export-daily",
		Name:     "Export",
		Kind:     ingjobs.KindHTTPCall,
		Schedule: "0 2 * * *",
		Enabled:  true,
		Params:   ingjobs.JobParams{URL: "http://127.0.0.1/export"},
	}
	if err := svc.CreateItem(item); err != nil {
		t.Fatal(err)
	}
	if err := svc.CreateItem(item); err == nil {
		t.Fatal("expected duplicate id error")
	}

	list, err := svc.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("items = %d", len(list.Items))
	}

	item.Name = "Export v2"
	item.Params.URL = "http://127.0.0.1/export/v2"
	if err := svc.UpdateItem("export-daily", item); err != nil {
		t.Fatal(err)
	}

	bad := item
	bad.Kind = ingjobs.KindScript
	if err := svc.UpdateItem("export-daily", bad); err == nil {
		t.Fatal("expected kind change error")
	}

	if err := svc.DeleteItem("export-daily"); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteItem("missing"); err == nil {
		t.Fatal("expected not found")
	}
}

func TestService_CreateItem_AutoGeneratesID(t *testing.T) {
	setupJobRunsDB(t)
	svc := newTestService(t, testIngressBase)

	item := ingjobs.Item{
		Name:     "Auto ID",
		Kind:     ingjobs.KindHTTPCall,
		Schedule: "0 2 * * *",
		Enabled:  true,
		Params:   ingjobs.JobParams{URL: "http://127.0.0.1/export"},
	}
	if err := svc.CreateItem(item); err != nil {
		t.Fatal(err)
	}
	list, err := svc.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("items = %d", len(list.Items))
	}
	got := list.Items[0].ID
	if got == "" {
		t.Fatal("expected generated id")
	}
	if _, err := uuid.Parse(got); err != nil {
		t.Fatalf("id %q is not a uuid: %v", got, err)
	}
}

func TestService_CreateItem_LegacyCommandKindNormalizes(t *testing.T) {
	setupJobRunsDB(t)
	svc := newTestService(t, testIngressBase)

	item := ingjobs.Item{
		Name:     "Legacy",
		Kind:     ingjobs.KindCommand,
		Schedule: "0 2 * * *",
		Enabled:  true,
		Params:   ingjobs.JobParams{Command: "/bin/echo", Args: []string{"ok"}},
	}
	if err := svc.CreateItem(item); err != nil {
		t.Fatal(err)
	}
	content, err := svc.ing.ReadYAML()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(content, "kind: script") {
		t.Fatalf("expected kind script in yaml, got:\n%s", content)
	}
	if !strings.Contains(content, "script:") {
		t.Fatalf("expected script content in yaml, got:\n%s", content)
	}
}

func TestService_UpdateBuiltin(t *testing.T) {
	svc := newTestService(t, testIngressBase)
	disabled := false
	if err := svc.UpdateBuiltin("purge_waf_events", BuiltinPatch{
		Enabled:  &disabled,
		Schedule: "0 6 * * *",
		Params:   &ingjobs.JobParams{RetainDays: 14},
	}); err != nil {
		t.Fatal(err)
	}
	list, err := svc.List()
	if err != nil {
		t.Fatal(err)
	}
	var found *JobView
	for i := range list.Builtins {
		if list.Builtins[i].ID == "purge_waf_events" {
			found = &list.Builtins[i]
			break
		}
	}
	if found == nil {
		t.Fatal("builtin not listed")
	}
	if found.Enabled {
		t.Fatal("expected disabled")
	}
	if found.Schedule != "0 6 * * *" || found.Params.RetainDays != 14 {
		t.Fatalf("builtin = %+v", found)
	}
}

func TestService_RunNow_HTTPPersistsResult(t *testing.T) {
	setupJobRunsDB(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Job", "1")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	}))
	t.Cleanup(srv.Close)

	svc := newTestService(t, testIngressBase)
	item := ingjobs.Item{
		ID:       "probe",
		Name:     "Probe",
		Kind:     ingjobs.KindHTTPCall,
		Schedule: "0 * * * *",
		Enabled:  true,
		Params:   ingjobs.JobParams{URL: srv.URL},
	}
	if err := svc.CreateItem(item); err != nil {
		t.Fatal(err)
	}

	row, err := svc.RunNow(SourceConfig, "probe")
	if err != nil {
		t.Fatal(err)
	}
	if row.Status != "success" {
		t.Fatalf("status = %q err=%q", row.Status, row.Error)
	}
	if row.Result == nil || row.Result.HTTP == nil {
		t.Fatal("missing http result on run row")
	}
	if row.Result.HTTP.Body != "hello" {
		t.Fatalf("body = %q", row.Result.HTTP.Body)
	}

	stored, err := svc.GetRun(row.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Result.HTTP.StatusCode != 200 {
		t.Fatalf("stored = %+v", stored.Result.HTTP)
	}
}

func TestService_RunNow_CommandPersistsLog(t *testing.T) {
	setupJobRunsDB(t)
	svc := newTestService(t, testIngressBase)
	item := ingjobs.Item{
		ID:       "echo-test",
		Name:     "Echo",
		Kind:     ingjobs.KindScript,
		Schedule: "0 * * * *",
		Enabled:  true,
		Params:   ingjobs.JobParams{Script: "echo scheduled", Shell: "sh"},
	}
	if err := svc.CreateItem(item); err != nil {
		t.Fatal(err)
	}

	row, err := svc.RunNow(SourceConfig, "echo-test")
	if err != nil {
		t.Fatal(err)
	}
	if row.Result == nil || row.Result.Command == nil {
		t.Fatal("missing command result")
	}
	if !strings.Contains(row.Result.Command.Log, "scheduled") {
		t.Fatalf("log = %q", row.Result.Command.Log)
	}
}

func TestService_RunNow_AlreadyRunning(t *testing.T) {
	setupJobRunsDB(t)
	svc := newTestService(t, testIngressBase)
	item := ingjobs.Item{
		ID:       "busy",
		Name:     "Busy",
		Kind:     ingjobs.KindHTTPCall,
		Schedule: "0 * * * *",
		Enabled:  true,
		Params:   ingjobs.JobParams{URL: "http://127.0.0.1/health"},
	}
	if err := svc.CreateItem(item); err != nil {
		t.Fatal(err)
	}
	svc.running["config:busy"] = true
	_, err := svc.RunNow(SourceConfig, "busy")
	if err == nil || !strings.Contains(err.Error(), "already running") {
		t.Fatalf("err = %v", err)
	}
}

func TestService_Reload_RegistersEnabledJobs(t *testing.T) {
	svc := newTestService(t, testIngressBase+`
jobs:
  items:
    - id: tick
      name: Tick
      kind: http_call
      schedule: "0 * * * *"
      enabled: true
      params:
        url: http://127.0.0.1/health
`)
	icfg, err := svc.ing.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(icfg.Jobs.Items) != 1 {
		t.Fatalf("config items = %d", len(icfg.Jobs.Items))
	}

	c := newFakeCron()
	if err := svc.Start(c); err != nil {
		t.Fatal(err)
	}
	if !c.HasJob(CronKey(SourceConfig, "tick")) {
		t.Fatal("expected custom job registered")
	}
	if spec, err := c.spec(CronKey(SourceConfig, "tick")); err != nil || spec != "0 * * * *" {
		t.Fatalf("tick spec = %q err=%v", spec, err)
	}
	for _, def := range AllBuiltins() {
		key := CronKey(SourceBuiltin, def.ID)
		if !c.HasJob(key) {
			t.Fatalf("expected builtin %s registered", def.ID)
		}
	}
}

func TestService_Capabilities(t *testing.T) {
	svc := newTestService(t, testIngressBase)
	caps := svc.Capabilities()
	if !caps.HTTPCall || !caps.Command || !caps.AllowCommand {
		t.Fatalf("caps = %+v", caps)
	}
	if !caps.CommandRestricted {
		t.Fatal("expected restricted when command_allowlist is set")
	}

	disabledYAML := `port: 8080
admin:
  enabled: true
  port: 9080
  jobs:
    allow_command: false
rules:
  - host: test.example.com
    backend:
      type: service
      service:
        name: upstream.local
        port: 8080
`
	svc2 := newTestService(t, disabledYAML)
	caps2 := svc2.Capabilities()
	if caps2.Command || caps2.AllowCommand {
		t.Fatalf("caps2 = %+v", caps2)
	}
}
