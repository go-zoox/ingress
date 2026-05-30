package jobs

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	ingjobs "github.com/go-zoox/ingress/core/jobs"
)

func TestMessageOutcome(t *testing.T) {
	out, err := messageOutcome("done", nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.Preview != "done" || out.Detail.Message != "done" {
		t.Fatalf("out = %+v", out)
	}

	out, err = messageOutcome("partial", errors.New("boom"))
	if err == nil {
		t.Fatal("expected error")
	}
	if out.Detail.Message != "partial" {
		t.Fatalf("message = %q", out.Detail.Message)
	}
}

func TestDefaultTimeoutSec(t *testing.T) {
	if DefaultTimeoutSec(0) != 60 {
		t.Fatal("expected default 60")
	}
	if DefaultTimeoutSec(30) != 30 {
		t.Fatal("expected 30")
	}
}

func TestService_OnFailureDisable(t *testing.T) {
	setupJobRunsDB(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	svc := newTestService(t, testIngressBase)
	item := ingjobs.Item{
		ID:         "fail-http",
		Name:       "Fail HTTP",
		Kind:       ingjobs.KindHTTPCall,
		Schedule:   "0 * * * *",
		Enabled:    true,
		OnFailure:  ingjobs.OnFailureDisable,
		TimeoutSec: 5,
		Params:     ingjobs.JobParams{URL: srv.URL},
	}
	if err := svc.CreateItem(item); err != nil {
		t.Fatal(err)
	}

	_, err := svc.RunNow(SourceConfig, "fail-http")
	if err == nil {
		t.Fatal("expected http failure")
	}

	list, err := svc.List()
	if err != nil {
		t.Fatal(err)
	}
	for _, j := range list.Items {
		if j.ID == "fail-http" && j.Enabled {
			t.Fatal("expected job disabled after on_failure=disable")
		}
	}
}
