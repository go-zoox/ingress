package jobs

import (
	"fmt"
	"sync"

	zcron "github.com/go-zoox/zoox/components/application/cron"
)

// fakeCron records scheduled jobs for unit tests.
type fakeCron struct {
	mu   sync.Mutex
	jobs map[string]string
}

var _ zcron.Cron = (*fakeCron)(nil)

func newFakeCron() *fakeCron {
	return &fakeCron{jobs: make(map[string]string)}
}

func (f *fakeCron) AddJob(id string, spec string, job func() error) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.jobs[id] = spec
	return nil
}

func (f *fakeCron) RemoveJob(id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.jobs, id)
	return nil
}

func (f *fakeCron) HasJob(id string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.jobs[id]
	return ok
}

func (f *fakeCron) ClearJobs() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.jobs = make(map[string]string)
	return nil
}

func (f *fakeCron) AddSecondlyJob(id string, cmd func() error) error {
	return f.AddJob(id, "@secondly", cmd)
}

func (f *fakeCron) AddMinutelyJob(id string, cmd func() error) error {
	return f.AddJob(id, "@minutely", cmd)
}

func (f *fakeCron) AddHourlyJob(id string, cmd func() error) error {
	return f.AddJob(id, "@hourly", cmd)
}

func (f *fakeCron) AddDailyJob(id string, cmd func() error) error {
	return f.AddJob(id, "@daily", cmd)
}

func (f *fakeCron) AddWeeklyJob(id string, cmd func() error) error {
	return f.AddJob(id, "@weekly", cmd)
}

func (f *fakeCron) AddMonthlyJob(id string, cmd func() error) error {
	return f.AddJob(id, "@monthly", cmd)
}

func (f *fakeCron) AddYearlyJob(id string, cmd func() error) error {
	return f.AddJob(id, "@yearly", cmd)
}

func (f *fakeCron) spec(id string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	spec, ok := f.jobs[id]
	if !ok {
		return "", fmt.Errorf("job %q not registered", id)
	}
	return spec, nil
}
