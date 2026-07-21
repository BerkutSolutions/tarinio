package services

import (
	"testing"
	"time"

	"waf/control-plane/internal/easysiteprofiles"
)

func TestEasySiteProfileService_MarkBasicAuthLoginUpdatesMatchingUserOnly(t *testing.T) {
	store := &fakeEasySiteProfileStore{items: map[string]easysiteprofiles.EasySiteProfile{
		"site-a": {SiteID: "site-a", SecurityAuthBasic: easysiteprofiles.SecurityAuthBasicSettings{Users: []easysiteprofiles.SecurityAuthUser{{Username: "alice", Enabled: true}, {Username: "disabled", Enabled: false}}}},
	}}
	service := NewEasySiteProfileService(store, &fakeSiteReader{}, &fakeEasyWAFStore{}, &fakeEasyAccessStore{}, &fakeEasyRateStore{}, nil, nil, nil)
	when := time.Date(2026, time.July, 21, 12, 0, 0, 0, time.UTC)
	if err := service.MarkBasicAuthLogin("site-a", "alice", when); err != nil {
		t.Fatalf("mark Basic Auth login: %v", err)
	}
	users := store.items["site-a"].SecurityAuthBasic.Users
	if users[0].LastLoginAt != when.Format(time.RFC3339) || users[1].LastLoginAt != "" {
		t.Fatalf("unexpected Basic Auth last-login state: %+v", users)
	}
	if err := service.MarkBasicAuthLogin("site-a", "disabled", when); err == nil {
		t.Fatal("disabled Basic Auth user must not receive a last-login timestamp")
	}
}
