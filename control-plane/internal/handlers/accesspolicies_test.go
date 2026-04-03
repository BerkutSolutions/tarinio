package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"waf/control-plane/internal/accesspolicies"
)

type fakeAccessPolicyService struct {
	items []accesspolicies.AccessPolicy
}

func (f *fakeAccessPolicyService) Create(ctx context.Context, item accesspolicies.AccessPolicy) (accesspolicies.AccessPolicy, error) {
	item.CreatedAt = "2026-04-01T00:00:00Z"
	item.UpdatedAt = "2026-04-01T00:00:00Z"
	f.items = append(f.items, item)
	return item, nil
}

func (f *fakeAccessPolicyService) List() ([]accesspolicies.AccessPolicy, error) {
	return append([]accesspolicies.AccessPolicy(nil), f.items...), nil
}

func (f *fakeAccessPolicyService) Update(ctx context.Context, item accesspolicies.AccessPolicy) (accesspolicies.AccessPolicy, error) {
	item.UpdatedAt = "2026-04-01T01:00:00Z"
	return item, nil
}

func (f *fakeAccessPolicyService) Delete(ctx context.Context, id string) error {
	return nil
}

func TestAccessPoliciesHandler_CreateAndList(t *testing.T) {
	handler := NewAccessPoliciesHandler(&fakeAccessPolicyService{})

	createReq := httptest.NewRequest(http.MethodPost, "/api/access-policies", bytes.NewBufferString(`{"id":"access-a","site_id":"site-a","enabled":true,"allowlist":["10.0.0.1","10.0.0.0/24"],"denylist":["192.168.1.1"]}`))
	createResp := httptest.NewRecorder()
	handler.ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", createResp.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/access-policies", nil)
	listResp := httptest.NewRecorder()
	handler.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", listResp.Code)
	}
}

func TestAccessPoliciesHandler_Delete(t *testing.T) {
	handler := NewAccessPoliciesHandler(&fakeAccessPolicyService{})
	req := httptest.NewRequest(http.MethodDelete, "/api/access-policies/access-a", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.Code)
	}
}
