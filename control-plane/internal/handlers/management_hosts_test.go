package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"waf/control-plane/internal/managementhosts"
	"waf/control-plane/internal/services"
	"waf/control-plane/internal/sites"
)

type managementHostsSiteReader struct{ items []sites.Site }

func (r managementHostsSiteReader) List() ([]sites.Site, error) {
	return append([]sites.Site(nil), r.items...), nil
}

func TestManagementHostsHandlerSetupAndOptimisticUpdate(t *testing.T) {
	store, err := managementhosts.NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewManagementHostsHandler(services.NewManagementHostsService(store, managementHostsSiteReader{items: []sites.Site{{ID: "panel", PrimaryHost: "panel.example", Enabled: true}}}, nil))
	get := httptest.NewRecorder()
	handler.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/api/settings/management-hosts", nil))
	if get.Code != http.StatusOK {
		t.Fatalf("get status = %d", get.Code)
	}
	var initial managementhosts.Settings
	if err := json.NewDecoder(get.Body).Decode(&initial); err != nil {
		t.Fatal(err)
	}
	if !initial.SetupRequired {
		t.Fatal("expected setup-required response")
	}
	put := httptest.NewRecorder()
	handler.ServeHTTP(put, httptest.NewRequest(http.MethodPut, "/api/settings/management-hosts", bytes.NewBufferString(`{"management_hosts":["PANEL.example."],"version":0}`)))
	if put.Code != http.StatusOK {
		t.Fatalf("put status = %d: %s", put.Code, put.Body.String())
	}
	stale := httptest.NewRecorder()
	handler.ServeHTTP(stale, httptest.NewRequest(http.MethodPut, "/api/settings/management-hosts", bytes.NewBufferString(`{"management_hosts":["next.example"],"version":0}`)))
	if stale.Code != http.StatusConflict {
		t.Fatalf("expected conflict status, got %d", stale.Code)
	}
	invalid := httptest.NewRecorder()
	handler.ServeHTTP(invalid, httptest.NewRequest(http.MethodPut, "/api/settings/management-hosts", bytes.NewBufferString(`{"management_hosts":["not a host"],"version":1}`)))
	if invalid.Code != http.StatusBadRequest || !strings.Contains(invalid.Body.String(), "invalid management host") {
		t.Fatalf("expected validation error, got %d: %s", invalid.Code, invalid.Body.String())
	}
}
