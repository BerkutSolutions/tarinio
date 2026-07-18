package handlers

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"waf/control-plane/internal/easysiteprofiles"
)

type fakeEasySiteProfileService struct {
	item          easysiteprofiles.EasySiteProfile
	err           error
	deletedSiteID string
}

func (f *fakeEasySiteProfileService) List() ([]easysiteprofiles.EasySiteProfile, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.item.SiteID == "" {
		f.item = easysiteprofiles.DefaultProfile("site-a")
	}
	return []easysiteprofiles.EasySiteProfile{f.item}, nil
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

func (f *fakeEasySiteProfileService) RevealAuthBasicPassword(_ context.Context, _ string, username string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	for _, user := range f.item.SecurityAuthBasic.Users {
		if user.Username == username {
			return user.Password, nil
		}
	}
	return "", fmt.Errorf("basic auth user %s not found", username)
}

func TestEasySiteProfilesHandler_List(t *testing.T) {
	handler := NewEasySiteProfilesHandler(&fakeEasySiteProfileService{})
	req := httptest.NewRequest(http.MethodGet, "/api/easy-site-profiles", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

func (f *fakeEasySiteProfileService) Upsert(ctx context.Context, profile easysiteprofiles.EasySiteProfile) (easysiteprofiles.EasySiteProfile, error) {
	if f.err != nil {
		return easysiteprofiles.EasySiteProfile{}, f.err
	}
	f.item = profile
	return profile, nil
}

func (f *fakeEasySiteProfileService) Delete(ctx context.Context, siteID string) error {
	if f.err != nil {
		return f.err
	}
	f.deletedSiteID = siteID
	return nil
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

func TestEasySiteProfilesHandler_RevealAuthPassword(t *testing.T) {
	service := &fakeEasySiteProfileService{item: easysiteprofiles.EasySiteProfile{SecurityAuthBasic: easysiteprofiles.SecurityAuthBasicSettings{Users: []easysiteprofiles.SecurityAuthUser{{Username: "admin", Password: "saved-secret"}}}}}
	handler := NewEasySiteProfilesHandler(service)
	req := httptest.NewRequest(http.MethodPost, "/api/easy-site-profiles/site-a/auth-password/reveal?username=admin", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), `"password":"saved-secret"`) {
		t.Fatalf("expected revealed password payload, got status=%d body=%s", resp.Code, resp.Body.String())
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

func TestEasySiteProfilesHandler_Delete(t *testing.T) {
	service := &fakeEasySiteProfileService{}
	handler := NewEasySiteProfilesHandler(service)
	req := httptest.NewRequest(http.MethodDelete, "/api/easy-site-profiles/site-a", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.Code)
	}
	if service.deletedSiteID != "site-a" {
		t.Fatalf("expected deleted site id to be recorded, got %q", service.deletedSiteID)
	}
}
