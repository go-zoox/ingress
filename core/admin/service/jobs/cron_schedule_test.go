package jobs

import (
	"testing"

	gocron "github.com/go-zoox/cron"
)

// Sub-minute presets must parse with the same cron engine ingress uses at runtime.
func TestSubMinuteCronSchedules(t *testing.T) {
	c, err := gocron.New()
	if err != nil {
		t.Fatal(err)
	}
	specs := []string{
		"@every 1s",
		"@every 10s",
		"@every 30s",
		"*/1 * * * *",
	}
	for _, spec := range specs {
		if err := c.AddJob("job-"+spec, spec, func() error { return nil }); err != nil {
			t.Fatalf("spec %q: %v", spec, err)
		}
	}
}
