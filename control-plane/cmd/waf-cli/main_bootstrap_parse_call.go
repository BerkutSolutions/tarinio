package main

import "flag"

func parseGlobalFlagSet(global *flag.FlagSet, argv []string) {
	global.Parse(argv)
}
