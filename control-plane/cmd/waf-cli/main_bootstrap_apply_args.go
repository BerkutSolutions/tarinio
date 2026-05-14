package main

func applyParsedArgs(opts globalOptions, args []string) globalOptions {
	opts.args = args
	return opts
}
