package services

import (
	"context"
	"testing"

	"waf/control-plane/internal/sites"
	"waf/control-plane/internal/wafpolicies"
)

type fakeWAFPolicyStore struct {
	items []wafpolicies.WAFPolicy
}

func (f *fakeWAFPolicyStore) Create(item wafpolicies.WAFPolicy) (wafpolicies.WAFPolicy, error) {
	f.items = append(f.items, item)
	return item, nil
}

func (f *fakeWAFPolicyStore) List() ([]wafpolicies.WAFPolicy, error) {
	return append([]wafpolicies.WAFPolicy(nil), f.items...), nil
}

func (f *fakeWAFPolicyStore) Update(item wafpolicies.WAFPolicy) (wafpolicies.WAFPolicy, error) {
	return item, nil
}

func (f *fakeWAFPolicyStore) Delete(id string) error {
	return nil
}

func TestWAFPolicyService_CreateValidatesSite(t *testing.T) {
	service := NewWAFPolicyService(
		&fakeWAFPolicyStore{},
		&fakeSiteReader{items: []sites.Site{{ID: "site-a"}}},
		nil,
	)

	created, err := service.Create(context.Background(), wafpolicies.WAFPolicy{
		ID:     "waf-a",
		SiteID: "site-a",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if created.ID != "waf-a" {
		t.Fatalf("unexpected policy: %+v", created)
	}
}

func TestWAFPolicyService_CreateRejectsMissingSite(t *testing.T) {
	service := NewWAFPolicyService(
		&fakeWAFPolicyStore{},
		&fakeSiteReader{},
		nil,
	)

	if _, err := service.Create(context.Background(), wafpolicies.WAFPolicy{ID: "waf-a", SiteID: "site-a"}); err == nil {
		t.Fatal("expected missing site error")
	}
}
