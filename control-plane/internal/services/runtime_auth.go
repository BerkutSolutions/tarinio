package services

import "net/http"

const runtimeAuthHeader = "X-WAF-Runtime-Token"

func setRuntimeAuthHeader(req *http.Request, token string) {
	if req == nil {
		return
	}
	if token == "" {
		return
	}
	req.Header.Set(runtimeAuthHeader, token)
}
