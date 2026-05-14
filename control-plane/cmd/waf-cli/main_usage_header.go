package main

import (
	"fmt"
	"os"
)

func printUsageHeader() {
	printUsageBanner()
}

func printUsageGlobalFlags() {
	fmt.Fprintln(os.Stderr, usageGlobalFlagsTitle())
	printUsageGlobalFlagLines()
	printUsageSectionSpacer()
}
