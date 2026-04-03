package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"waf/control-plane/internal/jobs"
)

type fakeRevisionApplyService struct{}

func (f *fakeRevisionApplyService) Apply(ctx context.Context, revisionID string) (jobs.Job, error) {
	return jobs.Job{
		ID:               "apply-" + revisionID,
		Type:             jobs.TypeApply,
		TargetRevisionID: revisionID,
		Status:           jobs.StatusSucceeded,
	}, nil
}

func TestRevisionApplyHandler_Apply(t *testing.T) {
	handler := NewRevisionApplyHandler(&fakeRevisionApplyService{})

	req := httptest.NewRequest(http.MethodPost, "/api/revisions/rev-000001/apply", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.Code)
	}
}
