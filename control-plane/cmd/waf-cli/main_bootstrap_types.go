package main

import "net/http"

const defaultBaseURL = "http://127.0.0.1:8080"

type cli struct {
	baseURL    string
	username   string
	password   string
	outputJSON bool
	client     *http.Client
	loggedIn   bool
}

type globalOptions struct {
	baseURL    string
	username   string
	password   string
	insecure   bool
	noAuth     bool
	outputJSON bool
	args       []string
}
