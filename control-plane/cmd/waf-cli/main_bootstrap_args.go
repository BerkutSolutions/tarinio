package main

import "flag"

func extractParsedArgs(global *flag.FlagSet) []string {
	return global.Args()
}
