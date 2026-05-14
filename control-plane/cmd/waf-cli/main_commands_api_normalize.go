package main

import "strings"

func normalizeAPITarget(methodArg, pathArg string) (string, string) {
	method := strings.ToUpper(strings.TrimSpace(methodArg))
	path := strings.TrimSpace(pathArg)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return method, path
}
