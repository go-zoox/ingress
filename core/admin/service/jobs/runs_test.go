package jobs

import (
	"testing"
	"time"
)

func TestCreateFinishAndGetRun_WithResultDetail(t *testing.T) {
	setupJobRunsDB(t)

	row, err := createRun("ping-job", SourceConfig, ingjobsKindHTTP, "manual")
	if err != nil {
		t.Fatal(err)
	}
	detail := RunResultDetail{
		HTTP: &HTTPRunResult{
			StatusCode: 200,
			Headers:    map[string]string{"Content-Type": "text/plain"},
			Body:       "pong",
		},
	}
	if err := finishRun(row, "success", 12.5, "HTTP 200", "", detail); err != nil {
		t.Fatal(err)
	}

	got, err := getRun(row.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "success" {
		t.Fatalf("status = %q", got.Status)
	}
	if got.Result == nil || got.Result.HTTP == nil {
		t.Fatal("expected http result")
	}
	if got.Result.HTTP.Body != "pong" {
		t.Fatalf("body = %q", got.Result.HTTP.Body)
	}
}

func TestListRuns_FiltersByJobID(t *testing.T) {
	setupJobRunsDB(t)

	for _, id := range []string{"alpha", "beta"} {
		row, err := createRun(id, SourceConfig, "http_call", "schedule")
		if err != nil {
			t.Fatal(err)
		}
		if err := finishRun(row, "success", 1, "ok", "", RunResultDetail{Message: id}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(time.Millisecond)
	}

	all, err := listRuns("", 10, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("all runs = %d", len(all))
	}

	alpha, err := listRuns("alpha", 10, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(alpha) != 1 || alpha[0].JobID != "alpha" {
		t.Fatalf("alpha runs = %+v", alpha)
	}

	withDetail, err := listRuns("alpha", 10, true)
	if err != nil {
		t.Fatal(err)
	}
	if withDetail[0].Result == nil || withDetail[0].Result.Message != "alpha" {
		t.Fatalf("result = %+v", withDetail[0].Result)
	}
}

func TestLastRunForJob_Empty(t *testing.T) {
	setupJobRunsDB(t)
	row, err := lastRunForJob("missing")
	if err != nil {
		t.Fatal(err)
	}
	if row != nil {
		t.Fatalf("expected nil, got %+v", row)
	}
}

const ingjobsKindHTTP = "http_call"
