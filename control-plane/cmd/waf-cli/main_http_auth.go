package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"
)

func newHTTPClient(insecure bool) (*http.Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	transport := &http.Transport{}
	if insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return &http.Client{
		Timeout:   20 * time.Second,
		Jar:       jar,
		Transport: transport,
	}, nil
}

func (c *cli) ensureLogin() error {
	if c.loggedIn {
		return nil
	}
	if strings.TrimSpace(c.username) == "" || strings.TrimSpace(c.password) == "" {
		return fmt.Errorf("username/password are required for auth commands")
	}
	payload := map[string]string{
		"username": c.username,
		"password": c.password,
	}
	status, body, err := c.rawRequest(http.MethodPost, "/api/auth/login", payload, false)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("login failed (%d): %s", status, extractErr(body))
	}
	c.loggedIn = true
	return nil
}
