package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"waf/control-plane/internal/jobs"
	"waf/control-plane/internal/revisions"
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

type fakeRevisionDeleteService struct {
	deleted string
}

type fakeRevisionApproveService struct{}

func (f *fakeRevisionApproveService) ApproveRevision(ctx context.Context, revisionID, comment string) (revisions.Revision, error) {
	return revisions.Revision{ID: revisionID, ApprovalStatus: revisions.ApprovalApproved}, nil
}

func (f *fakeRevisionDeleteService) Delete(ctx context.Context, revisionID string) error {
	f.deleted = revisionID
	return nil
}

func TestRevisionApplyHandler_Apply(t *testing.T) {
	handler := NewRevisionApplyHandler(&fakeRevisionApplyService{}, &fakeRevisionDeleteService{}, &fakeRevisionApproveService{})

	req := httptest.NewRequest(http.MethodPost, "/api/revisions/rev-000001/apply", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.Code)
	}
}

func TestRevisionApplyHandler_Delete(t *testing.T) {
	deleteService := &fakeRevisionDeleteService{}
	handler := NewRevisionApplyHandler(&fakeRevisionApplyService{}, deleteService, &fakeRevisionApproveService{})

	req := httptest.NewRequest(http.MethodDelete, "/api/revisions/rev-000001", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	if deleteService.deleted != "rev-000001" {
		t.Fatalf("expected revision delete to be requested, got %q", deleteService.deleted)
	}
}
