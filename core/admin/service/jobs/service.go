package jobs

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	zcron "github.com/go-zoox/zoox/components/application/cron"
	admincfg "github.com/go-zoox/ingress/core/admin/config"
	"github.com/go-zoox/ingress/core/admin/service"
	ingjobs "github.com/go-zoox/ingress/core/jobs"
	"github.com/go-zoox/logger"
	"gopkg.in/yaml.v3"
)

// JobView is the API representation of one scheduled job.
type JobView struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Source      string            `json:"source"`
	Kind        string            `json:"kind"`
	Description string            `json:"description,omitempty"`
	Schedule    string            `json:"schedule"`
	Enabled     bool              `json:"enabled"`
	TimeoutSec  int64             `json:"timeout_sec,omitempty"`
	OnFailure   string            `json:"on_failure,omitempty"`
	Params      ingjobs.JobParams `json:"params"`
	Deletable   bool              `json:"deletable"`
	Editable    bool              `json:"editable"`
	LastRun     *RunRow           `json:"last_run,omitempty"`
}

// Capabilities describes which custom job kinds are available.
type Capabilities struct {
	HTTPCall            bool     `json:"http_call"`
	Command             bool     `json:"command"`
	AllowCommand        bool     `json:"allow_command"`
	CommandRestricted   bool     `json:"command_restricted"`
	CommandAllowlist    []string `json:"command_allowlist,omitempty"`
	CommandReason       string   `json:"command_reason,omitempty"`
}

// ListResult is returned by GET /jobs.
type ListResult struct {
	Capabilities Capabilities `json:"capabilities"`
	Builtins     []JobView    `json:"builtins"`
	Items        []JobView    `json:"items"`
}

// Service manages scheduled jobs and zoox cron registration.
type Service struct {
	cfg     *admincfg.Config
	ing     *service.Ingress
	audit   *service.Audit
	tls     *service.TLS
	metrics *service.Metrics

	mu     sync.Mutex
	cron   zcron.Cron
	running map[string]bool
}

func New(cfg *admincfg.Config, ing *service.Ingress, audit *service.Audit, tls *service.TLS, metrics *service.Metrics) *Service {
	return &Service{
		cfg:     cfg,
		ing:     ing,
		audit:   audit,
		tls:     tls,
		metrics: metrics,
		running: make(map[string]bool),
	}
}

func (s *Service) Start(c zcron.Cron) error {
	if s == nil || c == nil {
		return fmt.Errorf("jobs service or cron is nil")
	}
	s.mu.Lock()
	s.cron = c
	s.mu.Unlock()
	return s.Reload()
}

func clearCronJobs(c zcron.Cron) error {
	if err := c.ClearJobs(); err != nil {
		// Zoox cron is lazy-started on first AddJob; initial Reload has nothing to clear.
		if strings.Contains(err.Error(), "not started yet") {
			return nil
		}
		return err
	}
	return nil
}

func (s *Service) Reload() error {
	s.mu.Lock()
	c := s.cron
	s.mu.Unlock()
	if c == nil {
		return nil
	}
	if err := clearCronJobs(c); err != nil {
		return fmt.Errorf("jobs: clear cron: %w", err)
	}
	icfg, err := s.ing.LoadConfig()
	if err != nil {
		return err
	}
	for _, def := range AllBuiltins() {
		var override *ingjobs.BuiltinOverride
		if icfg.Jobs.Builtins != nil {
			if o, ok := icfg.Jobs.Builtins[def.ID]; ok {
				copyO := o
				override = &copyO
			}
		}
		enabled, schedule, params := EffectiveBuiltin(def, override)
		if !enabled {
			continue
		}
		if err := s.registerJob(c, SourceBuiltin, def.ID, def.Kind, schedule, 0, ingjobs.OnFailureLog, params, icfg.Admin.Jobs); err != nil {
			logger.Warnf("jobs: register builtin %s: %v", def.ID, err)
		}
	}
	for _, item := range icfg.Jobs.Items {
		if !item.Enabled {
			continue
		}
		if err := ValidateCustomItem(&item, icfg.Admin.Jobs); err != nil {
			logger.Warnf("jobs: skip custom %s: %v", item.ID, err)
			continue
		}
		copyItem := item
		if err := s.registerJob(c, SourceConfig, copyItem.ID, copyItem.Kind, copyItem.Schedule, copyItem.TimeoutSec, DefaultOnFailure(copyItem.OnFailure), copyItem.Params, icfg.Admin.Jobs); err != nil {
			logger.Warnf("jobs: register custom %s: %v", copyItem.ID, err)
		}
	}
	return nil
}

func (s *Service) registerJob(c zcron.Cron, source, id, kind, schedule string, timeoutSec int64, onFailure string, params ingjobs.JobParams, policy ingjobs.AdminJobs) error {
	key := CronKey(source, id)
	jobID := id
	if source == SourceBuiltin {
		jobID = normalizeBuiltinID(id)
	}
	return c.AddJob(key, schedule, func() error {
		_, err := s.execute(context.Background(), source, jobID, kind, "schedule", timeoutSec, onFailure, params, policy)
		return err
	})
}

func (s *Service) Capabilities() Capabilities {
	icfg, err := s.ing.LoadConfig()
	if err != nil {
		return Capabilities{HTTPCall: true}
	}
	policy := icfg.Admin.Jobs
	enabled := policy.CommandExecutionEnabled()
	cap := Capabilities{
		HTTPCall:          true,
		Command:           enabled,
		AllowCommand:      enabled,
		CommandRestricted: policy.CommandRestricted(),
		CommandAllowlist:  append([]string(nil), policy.CommandAllowlist...),
	}
	if !enabled {
		cap.CommandReason = "已在 ingress.yaml 设置 admin.jobs.allow_command: false"
	} else if cap.CommandRestricted {
		cap.CommandReason = "仅允许 admin.jobs.command_allowlist 中的 Shell"
	}
	return cap
}

func (s *Service) List() (*ListResult, error) {
	icfg, err := s.ing.LoadConfig()
	if err != nil {
		return nil, err
	}
	out := &ListResult{Capabilities: s.Capabilities()}
	for _, def := range AllBuiltins() {
		var override *ingjobs.BuiltinOverride
		if icfg.Jobs.Builtins != nil {
			if o, ok := icfg.Jobs.Builtins[def.ID]; ok {
				copyO := o
				override = &copyO
			}
		}
		enabled, schedule, params := EffectiveBuiltin(def, override)
		last, _ := lastRunForJob(def.ID)
		out.Builtins = append(out.Builtins, JobView{
			ID:          def.ID,
			Name:        def.Name,
			Source:      SourceBuiltin,
			Kind:        def.Kind,
			Description: def.Description,
			Schedule:    schedule,
			Enabled:     enabled,
			Params:      params,
			Deletable:   false,
			Editable:    true,
			LastRun:     last,
		})
	}
	for _, item := range icfg.Jobs.Items {
		copyItem := item
		copyItem.Kind = ingjobs.NormalizeJobKind(copyItem.Kind)
		last, _ := lastRunForJob(copyItem.ID)
		out.Items = append(out.Items, JobView{
			ID:         copyItem.ID,
			Name:       copyItem.Name,
			Source:     SourceConfig,
			Kind:       copyItem.Kind,
			Schedule:   copyItem.Schedule,
			Enabled:    copyItem.Enabled,
			TimeoutSec: copyItem.TimeoutSec,
			OnFailure:  DefaultOnFailure(copyItem.OnFailure),
			Params:     copyItem.Params,
			Deletable:  true,
			Editable:   true,
			LastRun:    last,
		})
	}
	return out, nil
}

func (s *Service) RunNow(source, id string) (*RunRow, error) {
	icfg, err := s.ing.LoadConfig()
	if err != nil {
		return nil, err
	}
	policy := icfg.Admin.Jobs
	switch source {
	case SourceBuiltin:
		def, ok := BuiltinByID(id)
		if !ok {
			return nil, fmt.Errorf("unknown builtin job %q", id)
		}
		var override *ingjobs.BuiltinOverride
		if icfg.Jobs.Builtins != nil {
			if o, ok := icfg.Jobs.Builtins[def.ID]; ok {
				copyO := o
				override = &copyO
			}
		}
		_, schedule, params := EffectiveBuiltin(def, override)
		_ = schedule
		row, err := s.execute(context.Background(), SourceBuiltin, def.ID, def.Kind, "manual", 0, ingjobs.OnFailureLog, params, policy)
		return row, err
	case SourceConfig:
		for _, item := range icfg.Jobs.Items {
			if item.ID == id {
				if err := ValidateCustomItem(&item, policy); err != nil {
					return nil, err
				}
				return s.execute(context.Background(), SourceConfig, item.ID, item.Kind, "manual", item.TimeoutSec, DefaultOnFailure(item.OnFailure), item.Params, policy)
			}
		}
		return nil, fmt.Errorf("job %q not found", id)
	default:
		return nil, fmt.Errorf("invalid source %q", source)
	}
}

func (s *Service) execute(ctx context.Context, source, id, kind, trigger string, timeoutSec int64, onFailure string, params ingjobs.JobParams, policy ingjobs.AdminJobs) (*RunRow, error) {
	lockKey := source + ":" + id
	s.mu.Lock()
	if s.running[lockKey] {
		s.mu.Unlock()
		return nil, fmt.Errorf("job %q is already running", id)
	}
	s.running[lockKey] = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.running, lockKey)
		s.mu.Unlock()
	}()

	row, err := createRun(id, source, kind, trigger)
	if err != nil {
		return nil, err
	}

	timeout := DefaultTimeoutSec(timeoutSec)
	runCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	start := time.Now()
	outcome, runErr := s.runKind(runCtx, kind, params, policy, timeout)
	durationMs := float64(time.Since(start).Milliseconds())

	status := "success"
	errMsg := ""
	if runErr != nil {
		status = "failed"
		errMsg = runErr.Error()
	}
	_ = finishRun(row, status, durationMs, outcome.Preview, errMsg, outcome.Detail)
	if s.audit != nil {
		detail := fmt.Sprintf("job=%s source=%s kind=%s status=%s trigger=%s", id, source, kind, status, trigger)
		if errMsg != "" {
			detail += " error=" + errMsg
		}
		_ = s.audit.Record("job_run", detail, "scheduler")
	}
	if runErr != nil && onFailure == ingjobs.OnFailureDisable && source == SourceConfig {
		_ = s.setCustomEnabled(id, false)
		_ = s.Reload()
	}
	out := RunRow{
		ID:            row.ID,
		JobID:         row.JobID,
		Source:        row.Source,
		Kind:          row.Kind,
		Status:        status,
		DurationMs:    durationMs,
		OutputPreview: outcome.Preview,
		Result:        &outcome.Detail,
		Error:         errMsg,
		Trigger:       trigger,
		StartedAt:     row.StartedAt,
		FinishedAt:    time.Now(),
	}
	return &out, runErr
}

func (s *Service) runKind(ctx context.Context, kind string, params ingjobs.JobParams, policy ingjobs.AdminJobs, timeoutSec int64) (ExecOutcome, error) {
	switch ingjobs.NormalizeJobKind(kind) {
	case ingjobs.KindHTTPCall:
		return runHTTPCall(ctx, params, timeoutSec, policy)
	case ingjobs.KindScript:
		return runCommand(ctx, params, execEnv{timeoutSec: timeoutSec, policy: policy})
	case "purge_waf_events":
		msg, err := s.runPurgeWAFEvents(params)
		return messageOutcome(msg, err)
	case "purge_audit_logs":
		msg, err := s.runPurgeAuditLogs(params)
		return messageOutcome(msg, err)
	case "purge_metrics_buckets":
		msg, err := s.runPurgeMetricsBuckets(params)
		return messageOutcome(msg, err)
	case "check_tls_expiry":
		msg, err := s.runCheckTLSExpiry()
		return messageOutcome(msg, err)
	case "sync_geoip":
		service.SyncGeoIPFromIngress(s.ing)
		return messageOutcome("geoip sync triggered", nil)
	default:
		return ExecOutcome{}, fmt.Errorf("unsupported job kind %q", kind)
	}
}

func (s *Service) runPurgeWAFEvents(params ingjobs.JobParams) (string, error) {
	days := params.RetainDays
	if days <= 0 {
		days = 30
	}
	n, err := s.audit.PruneOldWAFEvents(fmt.Sprintf("%dh", days*24))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("deleted %d waf events older than %d days", n, days), nil
}

func (s *Service) runPurgeAuditLogs(params ingjobs.JobParams) (string, error) {
	days := params.RetainDays
	if days <= 0 {
		days = 90
	}
	n, err := s.audit.PruneOlderThan(days)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("deleted %d audit logs older than %d days", n, days), nil
}

func (s *Service) runPurgeMetricsBuckets(params ingjobs.JobParams) (string, error) {
	if s.metrics == nil {
		return "", fmt.Errorf("metrics service unavailable")
	}
	days := params.RetainDays
	if days <= 0 {
		days = 30
	}
	n, err := s.metrics.PurgePersistedBuckets(days)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("deleted %d metrics minute buckets older than %d days", n, days), nil
}

func (s *Service) runCheckTLSExpiry() (string, error) {
	if s.tls == nil {
		return "", fmt.Errorf("tls service unavailable")
	}
	rows, err := s.tls.List()
	if err != nil {
		return "", err
	}
	var warns []string
	for _, row := range rows {
		if row.Status == "expired" || row.Status == "expiring" {
			warns = append(warns, fmt.Sprintf("%s: %s (%d days)", row.Domain, row.Status, row.DaysRemaining))
		}
	}
	if len(warns) == 0 {
		return fmt.Sprintf("checked %d certificates: all ok", len(rows)), nil
	}
	return strings.Join(warns, "; "), fmt.Errorf("certificate warnings: %d", len(warns))
}

func (s *Service) UpdateBuiltin(id string, patch BuiltinPatch) error {
	def, ok := BuiltinByID(id)
	if !ok {
		return fmt.Errorf("unknown builtin job %q", id)
	}
	content, err := s.ing.ReadYAML()
	if err != nil {
		return err
	}
	jcfg, err := loadJobsFromContent(content)
	if err != nil {
		return err
	}
	if jcfg.Builtins == nil {
		jcfg.Builtins = map[string]ingjobs.BuiltinOverride{}
	}
	cur := jcfg.Builtins[def.ID]
	if patch.Enabled != nil {
		cur.Enabled = patch.Enabled
	}
	if strings.TrimSpace(patch.Schedule) != "" {
		cur.Schedule = strings.TrimSpace(patch.Schedule)
	}
	if patch.Params != nil {
		cur.Params = mergeParams(cur.Params, *patch.Params)
	}
	jcfg.Builtins[def.ID] = cur
	return s.writeJobsModule(content, jcfg)
}

type BuiltinPatch struct {
	Enabled  *bool              `json:"enabled"`
	Schedule string             `json:"schedule"`
	Params   *ingjobs.JobParams `json:"params"`
}

func (s *Service) CreateItem(item ingjobs.Item) error {
	item.ID = strings.TrimSpace(item.ID)
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	content, err := s.ing.ReadYAML()
	if err != nil {
		return err
	}
	icfg, err := s.ing.LoadConfigFromYAML(content)
	if err != nil {
		return err
	}
	if err := ValidateCustomItem(&item, icfg.Admin.Jobs); err != nil {
		return err
	}
	jcfg, err := loadJobsFromContent(content)
	if err != nil {
		return err
	}
	for _, existing := range jcfg.Items {
		if existing.ID == item.ID {
			return fmt.Errorf("job id %q already exists", item.ID)
		}
	}
	jcfg.Items = append(jcfg.Items, item)
	return s.writeJobsModule(content, jcfg)
}

func (s *Service) UpdateItem(id string, item ingjobs.Item) error {
	id = strings.TrimSpace(id)
	content, err := s.ing.ReadYAML()
	if err != nil {
		return err
	}
	icfg, err := s.ing.LoadConfigFromYAML(content)
	if err != nil {
		return err
	}
	item.ID = id
	item.Kind = ingjobs.NormalizeJobKind(item.Kind)
	if err := ValidateCustomItem(&item, icfg.Admin.Jobs); err != nil {
		return err
	}
	jcfg, err := loadJobsFromContent(content)
	if err != nil {
		return err
	}
	found := false
	for i := range jcfg.Items {
		if jcfg.Items[i].ID == id {
			if ingjobs.NormalizeJobKind(jcfg.Items[i].Kind) != item.Kind {
				return fmt.Errorf("job kind cannot be changed")
			}
			jcfg.Items[i] = item
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("job %q not found", id)
	}
	return s.writeJobsModule(content, jcfg)
}

func (s *Service) DeleteItem(id string) error {
	id = strings.TrimSpace(id)
	content, err := s.ing.ReadYAML()
	if err != nil {
		return err
	}
	jcfg, err := loadJobsFromContent(content)
	if err != nil {
		return err
	}
	next := make([]ingjobs.Item, 0, len(jcfg.Items))
	found := false
	for _, item := range jcfg.Items {
		if item.ID == id {
			found = true
			continue
		}
		next = append(next, item)
	}
	if !found {
		return fmt.Errorf("job %q not found", id)
	}
	jcfg.Items = next
	return s.writeJobsModule(content, jcfg)
}

func (s *Service) setCustomEnabled(id string, enabled bool) error {
	content, err := s.ing.ReadYAML()
	if err != nil {
		return err
	}
	jcfg, err := loadJobsFromContent(content)
	if err != nil {
		return err
	}
	for i := range jcfg.Items {
		if jcfg.Items[i].ID == id {
			jcfg.Items[i].Enabled = enabled
			return s.writeJobsModule(content, jcfg)
		}
	}
	return fmt.Errorf("job %q not found", id)
}

func (s *Service) writeJobsModule(content string, jcfg ingjobs.Config) error {
	moduleYAML, err := marshalJobsModule(jcfg)
	if err != nil {
		return err
	}
	merged, err := service.MergeConfigModule(content, "jobs", moduleYAML)
	if err != nil {
		return err
	}
	if err := s.ing.ValidateYAML(merged); err != nil {
		return err
	}
	if err := s.ing.WriteYAML(merged); err != nil {
		return err
	}
	return s.Reload()
}

func loadJobsFromContent(content string) (ingjobs.Config, error) {
	icfg, err := service.SplitConfigModules(content)
	if err != nil {
		return ingjobs.Config{}, err
	}
	for _, mod := range icfg {
		if mod.ID == "jobs" {
			if strings.TrimSpace(mod.YAML) == "" {
				return ingjobs.Config{}, nil
			}
			var wrapped struct {
				Jobs ingjobs.Config `yaml:"jobs"`
			}
			if err := yaml.Unmarshal([]byte(mod.YAML), &wrapped); err != nil {
				return ingjobs.Config{}, err
			}
			return wrapped.Jobs, nil
		}
	}
	return ingjobs.Config{}, nil
}

func marshalJobsModule(jcfg ingjobs.Config) (string, error) {
	if len(jcfg.Builtins) == 0 && len(jcfg.Items) == 0 {
		return "", nil
	}
	wrapped := struct {
		Jobs ingjobs.Config `yaml:"jobs"`
	}{Jobs: jcfg}
	b, err := yaml.Marshal(&wrapped)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (s *Service) ListRuns(jobID string, limit int) ([]RunRow, error) {
	return listRuns(jobID, limit, false)
}

func (s *Service) GetRun(id uint) (*RunRow, error) {
	return getRun(id)
}
