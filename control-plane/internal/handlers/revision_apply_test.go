package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"waf/control-plane/internal/auth"
	"waf/control-plane/internal/jobs"
	"waf/control-plane/internal/rbac"
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

type fakeFailedRevisionApplyService struct{}

func (f *fakeFailedRevisionApplyService) Apply(ctx context.Context, revisionID string) (jobs.Job, error) {
	return jobs.Job{
		ID:               "apply-" + revisionID,
		Type:             jobs.TypeApply,
		TargetRevisionID: revisionID,
		Status:           jobs.StatusFailed,
		Result:           "runtime reload failed",
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
	req = req.WithContext(auth.ContextWithSession(req.Context(), auth.SessionView{
		SessionID:   "s1",
		UserID:      "u1",
		Username:    "writer",
		Permissions: []string{string(rbac.PermissionRevisionsWrite)},
	}))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.Code)
	}
}

func TestRevisionApplyHandler_ApplyFailedJob(t *testing.T) {
	handler := NewRevisionApplyHandler(&fakeFailedRevisionApplyService{}, &fakeRevisionDeleteService{}, &fakeRevisionApproveService{})

	req := httptest.NewRequest(http.MethodPost, "/api/revisions/rev-000001/apply", nil)
	req = req.WithContext(auth.ContextWithSession(req.Context(), auth.SessionView{
		SessionID:   "s1",
		UserID:      "u1",
		Username:    "writer",
		Permissions: []string{string(rbac.PermissionRevisionsWrite)},
	}))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestRevisionApplyHandler_Delete(t *testing.T) {
	deleteService := &fakeRevisionDeleteService{}
	handler := NewRevisionApplyHandler(&fakeRevisionApplyService{}, deleteService, &fakeRevisionApproveService{})

	req := httptest.NewRequest(http.MethodDelete, "/api/revisions/rev-000001", nil)
	req = req.WithContext(auth.ContextWithSession(req.Context(), auth.SessionView{
		SessionID:   "s1",
		UserID:      "u1",
		Username:    "writer",
		Permissions: []string{string(rbac.PermissionRevisionsWrite)},
	}))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	if deleteService.deleted != "rev-000001" {
		t.Fatalf("expected revision delete to be requested, got %q", deleteService.deleted)
	}
}

func TestRevisionApplyHandler_ApproveRequiresApprovePermission(t *testing.T) {
	handler := NewRevisionApplyHandler(&fakeRevisionApplyService{}, &fakeRevisionDeleteService{}, &fakeRevisionApproveService{})
	req := httptest.NewRequest(http.MethodPost, "/api/revisions/rev-000001/approve", nil)
	req = req.WithContext(auth.ContextWithSession(req.Context(), auth.SessionView{
		SessionID:   "s1",
		UserID:      "u1",
		Username:    "operator",
		Permissions: []string{string(rbac.PermissionRevisionsWrite)},
	}))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.Code)
	}
}

func TestRevisionApplyHandler_ApproveWithPermission(t *testing.T) {
	handler := NewRevisionApplyHandler(&fakeRevisionApplyService{}, &fakeRevisionDeleteService{}, &fakeRevisionApproveService{})
	req := httptest.NewRequest(http.MethodPost, "/api/revisions/rev-000001/approve", nil)
	req = req.WithContext(auth.ContextWithSession(req.Context(), auth.SessionView{
		SessionID:   "s1",
		UserID:      "u1",
		Username:    "approver",
		Permissions: []string{string(rbac.PermissionRevisionsApprove)},
	}))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}
