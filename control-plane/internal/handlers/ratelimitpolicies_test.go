package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"waf/control-plane/internal/ratelimitpolicies"
)

type fakeRateLimitPolicyService struct {
	items []ratelimitpolicies.RateLimitPolicy
}

func (f *fakeRateLimitPolicyService) Create(ctx context.Context, item ratelimitpolicies.RateLimitPolicy) (ratelimitpolicies.RateLimitPolicy, error) {
	item.CreatedAt = "2026-04-01T00:00:00Z"
	item.UpdatedAt = "2026-04-01T00:00:00Z"
	f.items = append(f.items, item)
	return item, nil
}

func (f *fakeRateLimitPolicyService) List() ([]ratelimitpolicies.RateLimitPolicy, error) {
	return append([]ratelimitpolicies.RateLimitPolicy(nil), f.items...), nil
}

func (f *fakeRateLimitPolicyService) Update(ctx context.Context, item ratelimitpolicies.RateLimitPolicy) (ratelimitpolicies.RateLimitPolicy, error) {
	item.UpdatedAt = "2026-04-01T01:00:00Z"
	return item, nil
}

func (f *fakeRateLimitPolicyService) Delete(ctx context.Context, id string) error {
	return nil
}

func TestRateLimitPoliciesHandler_CreateAndList(t *testing.T) {
	handler := NewRateLimitPoliciesHandler(&fakeRateLimitPolicyService{})

	createReq := httptest.NewRequest(http.MethodPost, "/api/rate-limit-policies", bytes.NewBufferString(`{"id":"rate-a","site_id":"site-a","enabled":true,"limits":{"requests_per_second":10,"burst":20}}`))
	createResp := httptest.NewRecorder()
	handler.ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", createResp.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/rate-limit-policies", nil)
	listResp := httptest.NewRecorder()
	handler.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", listResp.Code)
	}
}

func TestRateLimitPoliciesHandler_Delete(t *testing.T) {
	handler := NewRateLimitPoliciesHandler(&fakeRateLimitPolicyService{})
	req := httptest.NewRequest(http.MethodDelete, "/api/rate-limit-policies/rate-a", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.Code)
	}
}
