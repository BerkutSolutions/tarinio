package main

import "fmt"

func dispatchUnknownCommand(args []string) error {
	return fmt.Errorf("unknown command: %s", args[0])
}
