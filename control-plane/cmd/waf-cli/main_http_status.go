package main

import "fmt"

func ensureHTTPSuccess(method, path string, status int, body []byte) error {
	if status < 200 || status >= 300 {
		return fmt.Errorf("%s %s failed (HTTP %d): %s", method, path, status, extractErr(body))
	}
	return nil
}
