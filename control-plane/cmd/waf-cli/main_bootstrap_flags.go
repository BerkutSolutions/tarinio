package main

import "flag"

func bindGlobalFlags(global *flag.FlagSet, opts *globalOptions) {
	global.StringVar(&opts.baseURL, "base-url", opts.baseURL, "control-plane base URL")
	global.StringVar(&opts.username, "username", opts.username, "control-plane username")
	global.StringVar(&opts.password, "password", opts.password, "control-plane password")
	global.BoolVar(&opts.insecure, "insecure", false, "skip TLS verification")
	global.BoolVar(&opts.noAuth, "no-auth", false, "skip auth for command")
	global.BoolVar(&opts.outputJSON, "json", false, "print raw JSON responses")
}
