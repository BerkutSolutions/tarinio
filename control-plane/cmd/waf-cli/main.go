package main

import (
	"os"
)

func main() {
	if err := runWafCLI(os.Args[1:]); err != nil {
		fatalf("%v", err)
	}
}


