package main

import (
	"net/http"
)

func (c *cli) rawRequest(method, path string, payload any, auth bool) (int, []byte, error) {
	if err := ensureRawRequestAuth(c, auth); err != nil {
		return 0, nil, err
	}
	bodyReader, err := buildJSONBodyReader(payload)
	if err != nil {
		return 0, nil, err
	}
	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return 0, nil, err
	}
	applyJSONContentType(req, payload)
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	return readHTTPResponseBody(resp)
}
