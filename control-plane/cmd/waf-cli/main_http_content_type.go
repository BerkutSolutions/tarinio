package main

import "net/http"

func applyJSONContentType(req *http.Request, payload any) {
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
}
