package main

import (
	"fmt"
	"os"
)

func printUsageSectionSpacer() {
	fmt.Fprintln(os.Stderr, usageBlankLine())
}
