package main

import "fmt"

func errCommandRequired() error {
	return fmt.Errorf("command is required")
}
