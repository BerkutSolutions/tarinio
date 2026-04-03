package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"waf/control-plane/internal/antiddos"
)

type fakeAntiDDoSService struct {
	item antiddos.Settings
}

func (f *fakeAntiDDoSService) Get() (antiddos.Settings, error) {
	if f.item.ConnLimit == 0 {
		f.item = antiddos.DefaultSettings()
	}
	return f.item, nil
}

func (f *fakeAntiDDoSService) Upsert(ctx context.Context, item antiddos.Settings) (antiddos.Settings, error) {
	f.item = antiddos.NormalizeSettings(item)
	return f.item, nil
}

func TestAntiDDoSHandler_GetPutAndPost(t *testing.T) {
	handler := NewAntiDDoSHandler(&fakeAntiDDoSService{})

	getReq := httptest.NewRequest(http.MethodGet, "/api/anti-ddos/settings", nil)
	getResp := httptest.NewRecorder()
	handler.ServeHTTP(getResp, getReq)
	if getResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", getResp.Code)
	}

	putReq := httptest.NewRequest(http.MethodPut, "/api/anti-ddos/settings", bytes.NewBufferString(`{"use_l4_guard":true,"chain_mode":"input","conn_limit":320,"rate_per_second":160,"rate_burst":320,"ports":[443],"target":"REJECT","enforce_l7_rate_limit":true,"l7_requests_per_second":40,"l7_burst":80,"l7_status_code":429}`))
	putResp := httptest.NewRecorder()
	handler.ServeHTTP(putResp, putReq)
	if putResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", putResp.Code)
	}

	postReq := httptest.NewRequest(http.MethodPost, "/api/anti-ddos/settings", bytes.NewBufferString(`{"use_l4_guard":false,"chain_mode":"auto","conn_limit":220,"rate_per_second":120,"rate_burst":220,"ports":[80,443],"target":"DROP","enforce_l7_rate_limit":false,"l7_requests_per_second":100,"l7_burst":200,"l7_status_code":429}`))
	postResp := httptest.NewRecorder()
	handler.ServeHTTP(postResp, postReq)
	if postResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", postResp.Code)
	}
}
