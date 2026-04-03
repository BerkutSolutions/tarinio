package services

import (
	"context"
	"testing"

	"waf/control-plane/internal/ratelimitpolicies"
	"waf/control-plane/internal/sites"
)

type fakeRateLimitPolicyStore struct {
	items []ratelimitpolicies.RateLimitPolicy
}

func (f *fakeRateLimitPolicyStore) Create(item ratelimitpolicies.RateLimitPolicy) (ratelimitpolicies.RateLimitPolicy, error) {
	f.items = append(f.items, item)
	return item, nil
}

func (f *fakeRateLimitPolicyStore) List() ([]ratelimitpolicies.RateLimitPolicy, error) {
	return append([]ratelimitpolicies.RateLimitPolicy(nil), f.items...), nil
}

func (f *fakeRateLimitPolicyStore) Update(item ratelimitpolicies.RateLimitPolicy) (ratelimitpolicies.RateLimitPolicy, error) {
	return item, nil
}

func (f *fakeRateLimitPolicyStore) Delete(id string) error {
	return nil
}

func TestRateLimitPolicyService_CreateValidatesSite(t *testing.T) {
	service := NewRateLimitPolicyService(
		&fakeRateLimitPolicyStore{},
		&fakeSiteReader{items: []sites.Site{{ID: "site-a"}}},
		nil,
	)

	created, err := service.Create(context.Background(), ratelimitpolicies.RateLimitPolicy{
		ID:      "rate-a",
		SiteID:  "site-a",
		Enabled: true,
		Limits: ratelimitpolicies.Limits{
			RequestsPerSecond: 10,
			Burst:             5,
		},
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if created.ID != "rate-a" {
		t.Fatalf("unexpected policy: %+v", created)
	}
}

func TestRateLimitPolicyService_CreateRejectsMissingSite(t *testing.T) {
	service := NewRateLimitPolicyService(
		&fakeRateLimitPolicyStore{},
		&fakeSiteReader{},
		nil,
	)

	if _, err := service.Create(context.Background(), ratelimitpolicies.RateLimitPolicy{ID: "rate-a", SiteID: "site-a"}); err == nil {
		t.Fatal("expected missing site error")
	}
}
