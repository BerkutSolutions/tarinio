package services

import (
	"net/http"
	"strings"
)

const runtimeAuthHeader = "X-WAF-Runtime-Token"
const legacyRuntimeAuthHeader = "X-WAF-Internal-Token"

func setRuntimeAuthHeader(req *http.Request, token string) {
	if req == nil {
		return
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return
	}
	req.Header.Set(runtimeAuthHeader, token)
	req.Header.Set(legacyRuntimeAuthHeader, token)
}
