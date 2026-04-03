package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"waf/control-plane/internal/easysiteprofiles"
)

type fakeEasySiteProfileService struct {
	item easysiteprofiles.EasySiteProfile
	err  error
}

func (f *fakeEasySiteProfileService) Get(siteID string) (easysiteprofiles.EasySiteProfile, error) {
	if f.err != nil {
		return easysiteprofiles.EasySiteProfile{}, f.err
	}
	if f.item.SiteID == "" {
		f.item = easysiteprofiles.DefaultProfile(siteID)
	}
	return f.item, nil
}

func (f *fakeEasySiteProfileService) Upsert(ctx context.Context, profile easysiteprofiles.EasySiteProfile) (easysiteprofiles.EasySiteProfile, error) {
	if f.err != nil {
		return easysiteprofiles.EasySiteProfile{}, f.err
	}
	f.item = profile
	return profile, nil
}

func TestEasySiteProfilesHandler_Get(t *testing.T) {
	handler := NewEasySiteProfilesHandler(&fakeEasySiteProfileService{})
	req := httptest.NewRequest(http.MethodGet, "/api/easy-site-profiles/site-a", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

func TestEasySiteProfilesHandler_Put(t *testing.T) {
	handler := NewEasySiteProfilesHandler(&fakeEasySiteProfileService{})
	body := bytes.NewBufferString(`{"front_service":{"server_name":"www.example.com","security_mode":"block","auto_lets_encrypt":true,"use_lets_encrypt_staging":false,"use_lets_encrypt_wildcard":false,"certificate_authority_server":"letsencrypt"}}`)
	req := httptest.NewRequest(http.MethodPut, "/api/easy-site-profiles/site-a", body)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

func TestEasySiteProfilesHandler_Post(t *testing.T) {
	handler := NewEasySiteProfilesHandler(&fakeEasySiteProfileService{})
	body := bytes.NewBufferString(`{"front_service":{"server_name":"www.example.com","security_mode":"block","auto_lets_encrypt":true,"use_lets_encrypt_staging":false,"use_lets_encrypt_wildcard":false,"certificate_authority_server":"letsencrypt"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/easy-site-profiles/site-a", body)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}
