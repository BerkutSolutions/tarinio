package easysiteprofiles

import "testing"

func TestNormalizeProfile_DoesNotReactivateDisabledBasicAuthUser(t *testing.T) {
	profile := DefaultProfile("site-a")
	profile.SecurityAuthBasic.UseAuthBasic = true
	profile.SecurityAuthBasic.AuthBasicUser = "disabled-user"
	profile.SecurityAuthBasic.AuthBasicPassword = "disabled-password"
	profile.SecurityAuthBasic.Users = []SecurityAuthUser{{Username: "disabled-user", Password: "disabled-password", Enabled: false}}

	normalized := normalizeProfile(profile)
	if len(normalized.SecurityAuthBasic.Users) != 1 || normalized.SecurityAuthBasic.Users[0].Enabled {
		t.Fatalf("disabled Basic Auth user must remain disabled after normalization: %+v", normalized.SecurityAuthBasic.Users)
	}
}
