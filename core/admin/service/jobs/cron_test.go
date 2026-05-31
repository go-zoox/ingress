package jobs

import (
	"testing"

	ingjobs "github.com/go-zoox/ingress/core/jobs"
	zcron "github.com/go-zoox/zoox/components/application/cron"
)

func TestZooxCron_ClearJobsAllowsScheduleReplace(t *testing.T) {
	c := zcron.New()
	key := "job:config:interval"
	if err := c.AddJob(key, "@every 1s", func() error { return nil }); err != nil {
		t.Fatalf("AddJob: %v", err)
	}
	if !c.HasJob(key) {
		t.Fatal("expected job registered")
	}
	if err := c.ClearJobs(); err != nil {
		t.Fatalf("ClearJobs: %v", err)
	}
	if c.HasJob(key) {
		t.Fatal("expected job cleared")
	}
	if err := c.AddJob(key, "@every 10s", func() error { return nil }); err != nil {
		t.Fatalf("AddJob after clear: %v", err)
	}
}

func TestService_Start_WithZooxCron(t *testing.T) {
	svc := newTestService(t, testIngressBase+`
jobs:
  items:
    - id: tick
      name: Tick
      kind: http_call
      schedule: "@every 1s"
      enabled: true
      params:
        url: http://127.0.0.1/health
`)
	c := zcron.New()
	if err := svc.Start(c); err != nil {
		t.Fatal(err)
	}
	key := CronKey(SourceConfig, "tick")
	if !c.HasJob(key) {
		t.Fatal("expected custom job registered on first Start")
	}
}

func TestService_Reload_ReplacesCustomSchedule(t *testing.T) {
	svc := newTestService(t, testIngressBase+`
jobs:
  items:
    - id: tick
      name: Tick
      kind: http_call
      schedule: "@every 1s"
      enabled: true
      params:
        url: http://127.0.0.1/health
`)
	c := newFakeCron()
	if err := svc.Start(c); err != nil {
		t.Fatal(err)
	}
	key := CronKey(SourceConfig, "tick")
	if spec, err := c.spec(key); err != nil || spec != "@every 1s" {
		t.Fatalf("initial spec = %q err=%v", spec, err)
	}

	item := ingjobs.Item{
		ID:       "tick",
		Name:     "Tick",
		Kind:     "http_call",
		Schedule: "@every 10s",
		Enabled:  true,
		Params:   ingjobs.JobParams{URL: "http://127.0.0.1/health"},
	}
	if err := svc.UpdateItem("tick", item); err != nil {
		t.Fatal(err)
	}
	if spec, err := c.spec(key); err != nil || spec != "@every 10s" {
		t.Fatalf("after update spec = %q err=%v", spec, err)
	}
}

func TestService_Reload_ReplacesCustomSchedule_ZooxCron(t *testing.T) {
	svc := newTestService(t, testIngressBase+`
jobs:
  items:
    - id: tick
      name: Tick
      kind: http_call
      schedule: "@every 1s"
      enabled: true
      params:
        url: http://127.0.0.1/health
`)
	c := zcron.New()
	if err := svc.Start(c); err != nil {
		t.Fatal(err)
	}
	key := CronKey(SourceConfig, "tick")
	if !c.HasJob(key) {
		t.Fatal("expected job registered")
	}

	item := ingjobs.Item{
		ID:       "tick",
		Name:     "Tick",
		Kind:     "http_call",
		Schedule: "@every 10s",
		Enabled:  true,
		Params:   ingjobs.JobParams{URL: "http://127.0.0.1/health"},
	}
	if err := svc.UpdateItem("tick", item); err != nil {
		t.Fatal(err)
	}
	if !c.HasJob(key) {
		t.Fatal("expected job re-registered after schedule update")
	}
	if err := c.AddJob(key, "@every 10s", func() error { return nil }); err == nil {
		t.Fatal("expected duplicate AddJob to fail, proving old entry was cleared")
	}
}
