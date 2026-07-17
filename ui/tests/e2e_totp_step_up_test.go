package tests

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestE2ETOTPStepUpProtectsCertificateExportAndLocksFailures(t *testing.T) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	if baseURL == "" {
		t.Skip("WAF_E2E_BASE_URL is not set; skipping TOTP step-up E2E")
	}
	adminClient, requestBaseURL, hostOverride := newE2EClientAndBase(t, baseURL)
	challengeURI := normalizeChallengeURI(strings.TrimSpace(os.Getenv("WAF_E2E_ANTIBOT_CHALLENGE_URI")))
	ensureManagementLoginAccess(t, adminClient, requestBaseURL, hostOverride, challengeURI)
	loginE2EStepUpUser(t, adminClient, requestBaseURL, hostOverride, "e2e-admin", "e2e-password-1234")

	username := "e2e_stepup_" + strconv.FormatInt(time.Now().UnixNano(), 36)
	password := "step-up-password-1234"
	create := postJSON(t, adminClient, requestBaseURL+"/api/administration/users", hostOverride, map[string]any{
		"id": username, "username": username, "email": username + "@example.test", "password": password, "role_ids": []string{"admin"},
	})
	if create.StatusCode != http.StatusCreated {
		t.Fatalf("create step-up user: status=%d body=%s", create.StatusCode, mustReadBody(t, create.Body))
	}
	_ = create.Body.Close()
	t.Cleanup(func() {
		resp := requestE2EJSON(t, adminClient, http.MethodDelete, requestBaseURL+"/api/administration/users/"+username, hostOverride, nil)
		_ = resp.Body.Close()
	})

	userClient, _, _ := newE2EClientAndBase(t, baseURL)
	ensureManagementLoginAccess(t, userClient, requestBaseURL, hostOverride, challengeURI)
	loginE2EStepUpUser(t, userClient, requestBaseURL, hostOverride, username, password)
	setup := postJSON(t, userClient, requestBaseURL+"/api/auth/2fa/setup", hostOverride, map[string]any{})
	if setup.StatusCode != http.StatusOK {
		t.Fatalf("TOTP setup: status=%d body=%s", setup.StatusCode, mustReadBody(t, setup.Body))
	}
	var setupBody struct {
		ChallengeID string `json:"challenge_id"`
		Secret      string `json:"secret"`
	}
	if err := json.NewDecoder(setup.Body).Decode(&setupBody); err != nil {
		t.Fatal(err)
	}
	if setupBody.ChallengeID == "" || setupBody.Secret == "" {
		t.Fatalf("invalid TOTP setup response: %+v", setupBody)
	}
	enable := postJSON(t, userClient, requestBaseURL+"/api/auth/2fa/enable", hostOverride, map[string]any{
		"challenge_id": setupBody.ChallengeID, "code": e2eTOTPCode(t, setupBody.Secret),
	})
	if enable.StatusCode != http.StatusOK {
		t.Fatalf("enable TOTP: status=%d body=%s", enable.StatusCode, mustReadBody(t, enable.Body))
	}
	_ = enable.Body.Close()
	logout := postJSON(t, userClient, requestBaseURL+"/api/auth/logout", hostOverride, map[string]any{})
	if logout.StatusCode != http.StatusNoContent {
		t.Fatalf("logout: status=%d body=%s", logout.StatusCode, mustReadBody(t, logout.Body))
	}
	_ = logout.Body.Close()

	login := postJSON(t, userClient, requestBaseURL+"/api/auth/login", hostOverride, map[string]any{"username": username, "password": password})
	var loginBody struct {
		RequiresTwoFactor bool   `json:"requires_2fa"`
		ChallengeID       string `json:"challenge_id"`
	}
	if login.StatusCode != http.StatusOK || json.NewDecoder(login.Body).Decode(&loginBody) != nil || !loginBody.RequiresTwoFactor || loginBody.ChallengeID == "" {
		t.Fatalf("TOTP login challenge: status=%d body=%s", login.StatusCode, mustReadBody(t, login.Body))
	}
	_ = login.Body.Close()
	finish := postJSON(t, userClient, requestBaseURL+"/api/auth/login/2fa", hostOverride, map[string]any{"challenge_id": loginBody.ChallengeID, "code": e2eTOTPCode(t, setupBody.Secret)})
	if finish.StatusCode != http.StatusOK {
		t.Fatalf("finish TOTP login: status=%d body=%s", finish.StatusCode, mustReadBody(t, finish.Body))
	}
	_ = finish.Body.Close()

	approvalRequest := postJSON(t, userClient, requestBaseURL+"/api/certificate-materials/export-approvals", hostOverride, map[string]any{"certificate_ids": []string{"missing-step-up-cert"}})
	if approvalRequest.StatusCode != http.StatusCreated {
		t.Fatalf("request export approval: status=%d body=%s", approvalRequest.StatusCode, mustReadBody(t, approvalRequest.Body))
	}
	var approval struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(approvalRequest.Body).Decode(&approval); err != nil || approval.ID == "" {
		t.Fatalf("decode export approval: id=%q err=%v", approval.ID, err)
	}
	approve := postJSON(t, adminClient, requestBaseURL+"/api/certificate-materials/export-approvals/"+approval.ID+"/approve", hostOverride, map[string]any{})
	if approve.StatusCode != http.StatusOK {
		t.Fatalf("approve export: status=%d body=%s", approve.StatusCode, mustReadBody(t, approve.Body))
	}
	_ = approve.Body.Close()

	exportURL := requestBaseURL + "/api/certificate-materials/export/missing-step-up-cert?approval_id=" + approval.ID
	beforeStepUp := getWithAuth(t, userClient, exportURL, hostOverride)
	if beforeStepUp.StatusCode != http.StatusForbidden || !strings.Contains(mustReadBody(t, beforeStepUp.Body), "fresh TOTP step-up") {
		t.Fatalf("export before step-up must be forbidden, got status=%d", beforeStepUp.StatusCode)
	}

	stepUp := postJSON(t, userClient, requestBaseURL+"/api/auth/step-up/totp", hostOverride, map[string]any{"code": e2eTOTPCode(t, setupBody.Secret)})
	if stepUp.StatusCode != http.StatusOK {
		t.Fatalf("complete step-up: status=%d body=%s", stepUp.StatusCode, mustReadBody(t, stepUp.Body))
	}
	_ = stepUp.Body.Close()
	afterStepUp := getWithAuth(t, userClient, exportURL, hostOverride)
	if afterStepUp.StatusCode != http.StatusNotFound {
		t.Fatalf("export must pass step-up and reach material lookup, got status=%d body=%s", afterStepUp.StatusCode, mustReadBody(t, afterStepUp.Body))
	}
	_ = afterStepUp.Body.Close()

	for attempt := 1; attempt <= 5; attempt++ {
		bad := postJSON(t, userClient, requestBaseURL+"/api/auth/step-up/totp", hostOverride, map[string]any{"code": "000000"})
		want := http.StatusUnauthorized
		if attempt == 5 {
			want = http.StatusTooManyRequests
		}
		if bad.StatusCode != want {
			t.Fatalf("bad step-up attempt %d: want=%d got=%d body=%s", attempt, want, bad.StatusCode, mustReadBody(t, bad.Body))
		}
		_ = bad.Body.Close()
	}
}

func loginE2EStepUpUser(t *testing.T, client *http.Client, baseURL, hostOverride, username, password string) {
	t.Helper()
	response := postJSON(t, client, baseURL+"/api/auth/login", hostOverride, map[string]any{"username": username, "password": password})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("login %s: status=%d body=%s", username, response.StatusCode, mustReadBody(t, response.Body))
	}
	_ = response.Body.Close()
}

func e2eTOTPCode(t *testing.T, secret string) string {
	t.Helper()
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(time.Now().UTC().Unix()/30))
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(buf)
	sum := mac.Sum(nil)
	offset := int(sum[len(sum)-1] & 0x0f)
	value := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", value%1_000_000)
}
