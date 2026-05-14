package main

import "net/http"

func (c *cli) requestJSON(method, path string, payload any, auth bool) (any, error) {
	if err := ensureRequestAuth(c, auth); err != nil {
		return nil, err
	}
	status, body, err := c.rawRequest(method, path, payload, false)
	if err != nil {
		return nil, err
	}
	if err := ensureHTTPSuccess(method, path, status, body); err != nil {
		return nil, err
	}
	if status == http.StatusNoContent {
		return nil, errHTTPNoContent()
	}
	return decodeJSONBody(body)
}

func (c *cli) requestNoContent(method, path string, payload any, auth bool) error {
	if err := ensureRequestAuth(c, auth); err != nil {
		return err
	}
	status, body, err := c.rawRequest(method, path, payload, false)
	if err != nil {
		return err
	}
	return ensureHTTPSuccess(method, path, status, body)
}
