package tests

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestE2EAPIFunctionalDeep(t *testing.T) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	if baseURL == "" {
		t.Skip("WAF_E2E_BASE_URL is not set; skipping deep functional e2e")
	}

	client, requestBaseURL, requestHostOverride := newE2EClientAndBase(t, baseURL)
	loginE2EUser(t, client, requestBaseURL, requestHostOverride)

	t.Run("SessionLogoutAndRelogin", func(t *testing.T) {
		logoutResp := postJSON(t, client, requestBaseURL+"/api/auth/logout", requestHostOverride, map[string]any{})
		assertStatusOneOfDeep(t, logoutResp, "/api/auth/logout", http.StatusOK, http.StatusNoContent)

		meResp := getWithAuth(t, client, requestBaseURL+"/api/auth/me", requestHostOverride)
		assertStatusOneOfDeep(t, meResp, "/api/auth/me after logout", http.StatusUnauthorized, http.StatusForbidden)

		loginE2EUser(t, client, requestBaseURL, requestHostOverride)
		meRespAfter := getWithAuth(t, client, requestBaseURL+"/api/auth/me", requestHostOverride)
		if meRespAfter.StatusCode != http.StatusOK {
			t.Fatalf("re-login check failed: status=%d body=%s", meRespAfter.StatusCode, mustReadBody(t, meRespAfter.Body))
		}
		_ = meRespAfter.Body.Close()
	})

	t.Run("AdminRolesAndUsersCRUD", func(t *testing.T) {
		roleID := "e2e-role-functional"
		userID := "e2e-user-functional"
		rolePayload := map[string]any{
			"id":          roleID,
			"name":        "E2E Functional Role",
			"permissions": []string{"sites.read", "sites.write", "auth.self"},
		}
		roleExisting := getWithAuth(t, client, requestBaseURL+"/api/administration/roles/"+roleID, requestHostOverride)
		if roleExisting.StatusCode == http.StatusNotFound {
			_ = roleExisting.Body.Close()
			roleCreate := postJSON(t, client, requestBaseURL+"/api/administration/roles", requestHostOverride, rolePayload)
			assertStatusOneOfDeep(t, roleCreate, "create role", http.StatusCreated, http.StatusOK)
		} else {
			assertStatusOneOfDeep(t, roleExisting, "get existing role", http.StatusOK)
			roleReset := requestDeepJSON(t, client, http.MethodPut, requestBaseURL+"/api/administration/roles/"+roleID, requestHostOverride, rolePayload)
			assertStatusOneOfDeep(t, roleReset, "reset existing role", http.StatusOK)
		}

		roleGet := getWithAuth(t, client, requestBaseURL+"/api/administration/roles/"+roleID, requestHostOverride)
		assertStatusOneOfDeep(t, roleGet, "get role", http.StatusOK)

		roleUpdate := requestDeepJSON(t, client, http.MethodPut, requestBaseURL+"/api/administration/roles/"+roleID, requestHostOverride, map[string]any{
			"name":        "E2E Functional Role Updated",
			"permissions": []string{"sites.read", "auth.self"},
		})
		assertStatusOneOfDeep(t, roleUpdate, "update role", http.StatusOK)

		userPayload := map[string]any{
			"id":         userID,
			"username":   userID,
			"email":      "e2e-functional@example.test",
			"password":   "password-123",
			"department": "QA",
			"position":   "Automation",
			"role_ids":   []string{roleID},
		}
		userExisting := getWithAuth(t, client, requestBaseURL+"/api/administration/users/"+userID, requestHostOverride)
		if userExisting.StatusCode == http.StatusNotFound {
			_ = userExisting.Body.Close()
			userCreate := postJSON(t, client, requestBaseURL+"/api/administration/users", requestHostOverride, userPayload)
			assertStatusOneOfDeep(t, userCreate, "create user", http.StatusCreated, http.StatusOK)
		} else {
			assertStatusOneOfDeep(t, userExisting, "get existing user", http.StatusOK)
			userReset := requestDeepJSON(t, client, http.MethodPut, requestBaseURL+"/api/administration/users/"+userID, requestHostOverride, userPayload)
			assertStatusOneOfDeep(t, userReset, "reset existing user", http.StatusOK)
		}

		userGet := getWithAuth(t, client, requestBaseURL+"/api/administration/users/"+userID, requestHostOverride)
		assertStatusOneOfDeep(t, userGet, "get user", http.StatusOK)

		userUpdate := requestDeepJSON(t, client, http.MethodPut, requestBaseURL+"/api/administration/users/"+userID, requestHostOverride, map[string]any{
			"email":      "e2e-functional-updated@example.test",
			"department": "QA-2",
			"position":   "Automation-2",
			"role_ids":   []string{roleID},
			"is_active":  true,
		})
		assertStatusOneOfDeep(t, userUpdate, "update user", http.StatusOK)

		usersList := getWithAuth(t, client, requestBaseURL+"/api/administration/users", requestHostOverride)
		if usersList.StatusCode != http.StatusOK {
			t.Fatalf("list users failed: status=%d body=%s", usersList.StatusCode, mustReadBody(t, usersList.Body))
		}
		var usersPayload map[string]any
		if err := json.NewDecoder(usersList.Body).Decode(&usersPayload); err != nil {
			t.Fatalf("decode users list: %v", err)
		}
		if _, ok := usersPayload["users"]; !ok {
			t.Fatalf("users list missing users field: %#v", usersPayload)
		}

		roleList := getWithAuth(t, client, requestBaseURL+"/api/administration/roles", requestHostOverride)
		if roleList.StatusCode != http.StatusOK {
			t.Fatalf("list roles failed: status=%d body=%s", roleList.StatusCode, mustReadBody(t, roleList.Body))
		}
		var roleListPayload map[string]any
		if err := json.NewDecoder(roleList.Body).Decode(&roleListPayload); err != nil {
			t.Fatalf("decode roles list: %v", err)
		}
		if _, ok := roleListPayload["roles"]; !ok {
			t.Fatalf("roles list missing roles field: %#v", roleListPayload)
		}
	})

	t.Run("EventsRequestsAuditProbes", func(t *testing.T) {
		eventsProbe := getWithAuth(t, client, requestBaseURL+"/api/events?probe=1", requestHostOverride)
		assertStatusOneOfDeep(t, eventsProbe, "/api/events probe", http.StatusOK, http.StatusBadGateway)

		requestsProbe := getWithAuth(t, client, requestBaseURL+"/api/requests?probe=1", requestHostOverride)
		assertStatusOneOfDeep(t, requestsProbe, "/api/requests probe", http.StatusOK, http.StatusBadGateway)

		eventsData := getWithAuth(t, client, requestBaseURL+"/api/events", requestHostOverride)
		if eventsData.StatusCode != http.StatusOK {
			t.Fatalf("/api/events failed: status=%d body=%s", eventsData.StatusCode, mustReadBody(t, eventsData.Body))
		}
		var eventsPayload map[string]any
		if err := json.NewDecoder(eventsData.Body).Decode(&eventsPayload); err != nil {
			t.Fatalf("decode /api/events: %v", err)
		}
		if _, ok := eventsPayload["events"]; !ok {
			t.Fatalf("/api/events missing events field: %#v", eventsPayload)
		}

		requestsData := getWithAuth(t, client, requestBaseURL+"/api/requests?limit=5&offset=0", requestHostOverride)
		if requestsData.StatusCode != http.StatusOK {
			t.Fatalf("/api/requests failed: status=%d body=%s", requestsData.StatusCode, mustReadBody(t, requestsData.Body))
		}
		var requestsPayload []map[string]any
		if err := json.NewDecoder(requestsData.Body).Decode(&requestsPayload); err != nil {
			t.Fatalf("decode /api/requests: %v", err)
		}

		auditData := getWithAuth(t, client, requestBaseURL+"/api/audit?limit=5&offset=0", requestHostOverride)
		if auditData.StatusCode != http.StatusOK {
			t.Fatalf("/api/audit failed: status=%d body=%s", auditData.StatusCode, mustReadBody(t, auditData.Body))
		}
		var auditPayload map[string]any
		if err := json.NewDecoder(auditData.Body).Decode(&auditPayload); err != nil {
			t.Fatalf("decode /api/audit: %v", err)
		}
		if _, ok := auditPayload["items"]; !ok {
			t.Fatalf("/api/audit missing items field: %#v", auditPayload)
		}
	})

	t.Run("RevisionStatusesClear", func(t *testing.T) {
		clearResp := requestDeepJSON(t, client, http.MethodDelete, requestBaseURL+"/api/revisions/statuses", requestHostOverride, nil)
		if clearResp.StatusCode != http.StatusOK {
			t.Fatalf("clear revision statuses failed: status=%d body=%s", clearResp.StatusCode, mustReadBody(t, clearResp.Body))
		}
		var payload map[string]any
		if err := json.NewDecoder(clearResp.Body).Decode(&payload); err != nil {
			t.Fatalf("decode revision statuses clear response: %v", err)
		}
		if cleared, _ := payload["cleared"].(bool); !cleared {
			t.Fatalf("expected cleared=true, got %#v", payload)
		}
	})
}

func requestDeepJSON(t *testing.T, client *http.Client, method, endpoint, hostOverride string, payload any) *http.Response {
	t.Helper()
	if payload == nil {
		req, err := http.NewRequest(method, endpoint, nil)
		if err != nil {
			t.Fatalf("create %s %s request: %v", method, endpoint, err)
		}
		req.Header.Set("Accept", "application/json")
		if hostOverride != "" {
			req.Host = hostOverride
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request %s %s failed: %v", method, endpoint, err)
		}
		return resp
	}
	return requestE2EJSON(t, client, method, endpoint, hostOverride, payload)
}

func assertStatusOneOfDeep(t *testing.T, resp *http.Response, action string, allowed ...int) {
	t.Helper()
	for _, code := range allowed {
		if resp.StatusCode == code {
			_ = resp.Body.Close()
			return
		}
	}
	t.Fatalf("%s failed: status=%d body=%s", action, resp.StatusCode, mustReadBody(t, resp.Body))
}
