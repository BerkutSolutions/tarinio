package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"
)

const basicAuthVerifyPath = "/auth/verify/basic"

func reportBasicAuthLogin(item parsedAccess) {
	if item.status != http.StatusNoContent || normalizeBurstPath(item.path) != basicAuthVerifyPath || item.siteID == "" || item.remoteUser == "" {
		return
	}
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_CONTROL_PLANE_INTERNAL_URL")), "/")
	token := strings.TrimSpace(os.Getenv("WAF_RUNTIME_API_TOKEN"))
	if baseURL == "" || token == "" {
		return
	}
	payload, err := json.Marshal(map[string]string{"site_id": item.siteID, "username": item.remoteUser, "occurred_at": item.when.UTC().Format(time.RFC3339)})
	if err != nil {
		return
	}
	request, err := http.NewRequest(http.MethodPost, baseURL+"/api/internal/runtime/basic-auth-login", bytes.NewReader(payload))
	if err != nil {
		return
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(runtimeAuthHeader, token)
	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Do(request)
	if err == nil && response != nil {
		_ = response.Body.Close()
	}
}
