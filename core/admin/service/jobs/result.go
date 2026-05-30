package jobs

// RunResultDetail is persisted as JSON on job_run.result_detail.
type RunResultDetail struct {
	HTTP    *HTTPRunResult    `json:"http,omitempty"`
	Command *CommandRunResult `json:"command,omitempty"`
	Message string            `json:"message,omitempty"`
}

// HTTPRunResult captures an http_call execution response.
type HTTPRunResult struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

// CommandRunResult captures command stdout/stderr output.
type CommandRunResult struct {
	Log string `json:"log"`
}

// ExecOutcome is the structured result of one job execution.
type ExecOutcome struct {
	Preview string
	Detail  RunResultDetail
}
