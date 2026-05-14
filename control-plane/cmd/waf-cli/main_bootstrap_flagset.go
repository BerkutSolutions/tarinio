package main

import "flag"

func newGlobalFlagSet(opts *globalOptions) *flag.FlagSet {
	global := flag.NewFlagSet("waf-cli", flag.ExitOnError)
	bindGlobalFlags(global, opts)
	global.Usage = usage
	return global
}
