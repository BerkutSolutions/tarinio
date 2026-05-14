package main

import "net/http"

func newCLI(baseURL, username, password string, outputJSON bool, client *http.Client) *cli {
	return &cli{
		baseURL:    baseURL,
		username:   username,
		password:   password,
		outputJSON: outputJSON,
		client:     client,
	}
}
