package main

import (
	"fmt"
	"os"
)

func printUsageStderrLine(text string) {
	fmt.Fprintln(os.Stderr, text)
}
