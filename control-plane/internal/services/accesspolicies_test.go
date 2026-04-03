package services

import (
	"context"
	"testing"

	"waf/control-plane/internal/accesspolicies"
	"waf/control-plane/internal/sites"
)

type fakeAccessPolicyStore struct {
	items []accesspolicies.AccessPolicy
}

func (f *fakeAccessPolicyStore) Create(item accesspolicies.AccessPolicy) (accesspolicies.AccessPolicy, error) {
	f.items = append(f.items, item)
	return item, nil
}

func (f *fakeAccessPolicyStore) List() ([]accesspolicies.AccessPolicy, error) {
	return append([]accesspolicies.AccessPolicy(nil), f.items...), nil
}

func (f *fakeAccessPolicyStore) Update(item accesspolicies.AccessPolicy) (accesspolicies.AccessPolicy, error) {
	return item, nil
}

func (f *fakeAccessPolicyStore) Delete(id string) error {
	return nil
}

func TestAccessPolicyService_CreateValidatesSite(t *testing.T) {
	service := NewAccessPolicyService(
		&fakeAccessPolicyStore{},
		&fakeSiteReader{items: []sites.Site{{ID: "site-a"}}},
		nil,
	)

	created, err := service.Create(context.Background(), accesspolicies.AccessPolicy{
		ID:     "access-a",
		SiteID: "site-a",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if created.ID != "access-a" {
		t.Fatalf("unexpected policy: %+v", created)
	}
}

func TestAccessPolicyService_CreateRejectsMissingSite(t *testing.T) {
	service := NewAccessPolicyService(
		&fakeAccessPolicyStore{},
		&fakeSiteReader{},
		nil,
	)

	if _, err := service.Create(context.Background(), accesspolicies.AccessPolicy{ID: "access-a", SiteID: "site-a"}); err == nil {
		t.Fatal("expected missing site error")
	}
}
