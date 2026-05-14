package main

import "fmt"

func errHTTPNoContent() error {
	return fmt.Errorf("HTTP 204")
}
