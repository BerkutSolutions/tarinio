package main

import (
	"io"
	"net/http"
)

func readHTTPResponseBody(resp *http.Response) (int, []byte, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, body, nil
}
