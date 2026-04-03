package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"waf/control-plane/internal/wafpolicies"
)

type fakeWAFPolicyService struct {
	items []wafpolicies.WAFPolicy
}

func (f *fakeWAFPolicyService) Create(ctx context.Context, item wafpolicies.WAFPolicy) (wafpolicies.WAFPolicy, error) {
	item.CreatedAt = "2026-04-01T00:00:00Z"
	item.UpdatedAt = "2026-04-01T00:00:00Z"
	f.items = append(f.items, item)
	return item, nil
}

func (f *fakeWAFPolicyService) List() ([]wafpolicies.WAFPolicy, error) {
	return append([]wafpolicies.WAFPolicy(nil), f.items...), nil
}

func (f *fakeWAFPolicyService) Update(ctx context.Context, item wafpolicies.WAFPolicy) (wafpolicies.WAFPolicy, error) {
	item.UpdatedAt = "2026-04-01T01:00:00Z"
	return item, nil
}

func (f *fakeWAFPolicyService) Delete(ctx context.Context, id string) error {
	return nil
}

func TestWAFPoliciesHandler_CreateAndList(t *testing.T) {
	handler := NewWAFPoliciesHandler(&fakeWAFPolicyService{})

	createReq := httptest.NewRequest(http.MethodPost, "/api/waf-policies", bytes.NewBufferString(`{"id":"waf-a","site_id":"site-a","enabled":true,"mode":"detection","crs_enabled":true,"custom_rule_includes":["rules/a.conf"],"rule_overrides":[{"rule_id":"941100","enabled":false}]}`))
	createResp := httptest.NewRecorder()
	handler.ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", createResp.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/waf-policies", nil)
	listResp := httptest.NewRecorder()
	handler.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", listResp.Code)
	}
}

func TestWAFPoliciesHandler_Delete(t *testing.T) {
	handler := NewWAFPoliciesHandler(&fakeWAFPolicyService{})
	req := httptest.NewRequest(http.MethodDelete, "/api/waf-policies/waf-a", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.Code)
	}
}
