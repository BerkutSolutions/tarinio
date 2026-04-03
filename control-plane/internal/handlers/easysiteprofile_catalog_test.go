package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEasySiteProfileCatalogHandler_GetCountries(t *testing.T) {
	handler := NewEasySiteProfileCatalogHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/easy-site-profiles/catalog/countries", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	body := resp.Body.String()
	if !strings.Contains(body, "\"continents\"") || !strings.Contains(body, "\"countries\"") {
		t.Fatalf("unexpected response payload: %s", body)
	}
}

func TestEasySiteProfileCatalogHandler_NotFound(t *testing.T) {
	handler := NewEasySiteProfileCatalogHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/easy-site-profiles/catalog", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.Code)
	}
}
