package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"waf/control-plane/internal/services"
)

type fakeRevisionCatalogService struct{}

func (f *fakeRevisionCatalogService) List(ctx context.Context) (services.RevisionCatalogResponse, error) {
	return services.RevisionCatalogResponse{
		Services: []services.RevisionServiceCard{{SiteID: "site-a"}},
	}, nil
}

func TestRevisionCatalogHandler_List(t *testing.T) {
	handler := NewRevisionCatalogHandler(&fakeRevisionCatalogService{})

	req := httptest.NewRequest(http.MethodGet, "/api/revisions", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}
