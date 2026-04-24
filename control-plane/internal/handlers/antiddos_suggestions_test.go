package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"waf/control-plane/internal/antiddossuggestions"
)

type fakeAntiDDoSSuggestionsService struct {
	items map[string]antiddossuggestions.Suggestion
}

func (f *fakeAntiDDoSSuggestionsService) List() ([]antiddossuggestions.Suggestion, error) {
	out := make([]antiddossuggestions.Suggestion, 0, len(f.items))
	for _, item := range f.items {
		out = append(out, item)
	}
	return out, nil
}

func (f *fakeAntiDDoSSuggestionsService) Upsert(ctx context.Context, item antiddossuggestions.Suggestion) (antiddossuggestions.Suggestion, error) {
	if f.items == nil {
		f.items = map[string]antiddossuggestions.Suggestion{}
	}
	item = antiddossuggestions.NormalizeSuggestion(item)
	f.items[item.ID] = item
	return item, nil
}

func (f *fakeAntiDDoSSuggestionsService) SetStatus(ctx context.Context, id, status string) (antiddossuggestions.Suggestion, error) {
	item := f.items[id]
	item.Status = status
	f.items[id] = item
	return item, nil
}

func TestAntiDDoSRuleSuggestionsHandler_ListUpsertStatus(t *testing.T) {
	handler := NewAntiDDoSRuleSuggestionsHandler(&fakeAntiDDoSSuggestionsService{
		items: map[string]antiddossuggestions.Suggestion{},
	})

	postReq := httptest.NewRequest(http.MethodPost, "/api/anti-ddos/rule-suggestions", bytes.NewBufferString(`{"path_prefix":"/.env","hits":42,"unique_ips":11}`))
	postResp := httptest.NewRecorder()
	handler.ServeHTTP(postResp, postReq)
	if postResp.Code != http.StatusOK {
		t.Fatalf("expected 200 for upsert, got %d", postResp.Code)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/anti-ddos/rule-suggestions", nil)
	getResp := httptest.NewRecorder()
	handler.ServeHTTP(getResp, getReq)
	if getResp.Code != http.StatusOK {
		t.Fatalf("expected 200 for list, got %d", getResp.Code)
	}

	statusReq := httptest.NewRequest(http.MethodPost, "/api/anti-ddos/rule-suggestions/path-.env/status", bytes.NewBufferString(`{"status":"shadow"}`))
	statusResp := httptest.NewRecorder()
	handler.ServeHTTP(statusResp, statusReq)
	if statusResp.Code != http.StatusOK {
		t.Fatalf("expected 200 for status update, got %d", statusResp.Code)
	}
}
