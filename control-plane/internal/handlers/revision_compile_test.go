package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"waf/control-plane/internal/jobs"
	"waf/control-plane/internal/revisions"
	"waf/control-plane/internal/services"
)

type fakeRevisionCompileService struct{}

func (f *fakeRevisionCompileService) Create(ctx context.Context) (services.CompileRequestResult, error) {
	return services.CompileRequestResult{
		Revision: revisions.Revision{ID: "rev-000001", Version: 1, Status: revisions.StatusPending},
		Job:      jobs.Job{ID: "compile-rev-000001", Type: jobs.TypeCompile, TargetRevisionID: "rev-000001", Status: jobs.StatusPending},
	}, nil
}

func TestRevisionCompileHandler_Create(t *testing.T) {
	handler := NewRevisionCompileHandler(&fakeRevisionCompileService{})

	req := httptest.NewRequest(http.MethodPost, "/api/revisions/compile", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.Code)
	}
}
