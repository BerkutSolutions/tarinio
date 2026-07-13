package services

import (
	"context"
	"strings"
	"testing"

	"waf/control-plane/internal/managementhosts"
	"waf/control-plane/internal/sites"
)

type managementHostTestSites struct{ items []sites.Site }

func (r managementHostTestSites) List() ([]sites.Site, error) {
	return append([]sites.Site(nil), r.items...), nil
}

func TestManagementHostsUpdateRequiresEnabledMatchingSiteAndPreservesPreviousValue(t *testing.T) {
	store, err := managementhosts.NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	service := NewManagementHostsService(store, managementHostTestSites{items: []sites.Site{{ID: "panel", PrimaryHost: "panel.example", Enabled: true}}}, nil)
	created, err := service.Update(context.Background(), []string{"panel.example"}, 0)
	if err != nil {
		t.Fatalf("save valid management host: %v", err)
	}
	if _, err := service.Update(context.Background(), []string{"missing.example"}, created.Version); err == nil || !strings.Contains(err.Error(), "must match an enabled") {
		t.Fatalf("expected enabled site validation error, got %v", err)
	}
	after, err := store.Get()
	if err != nil {
		t.Fatal(err)
	}
	if after.Version != created.Version || len(after.Hosts) != 1 || after.Hosts[0] != "panel.example" {
		t.Fatalf("failed update changed persisted safe value: %+v", after)
	}
}
