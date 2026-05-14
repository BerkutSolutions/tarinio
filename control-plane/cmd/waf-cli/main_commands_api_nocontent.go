package main

import (
	"fmt"
	"strings"
)

func handleAPINoContentError(err error, method, path string) bool {
	if err == nil {
		return false
	}
	if strings.Contains(err.Error(), "HTTP 204") {
		fmt.Printf("%s %s -> 204 No Content\n", method, path)
		return true
	}
	return false
}
