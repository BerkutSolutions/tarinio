package main

func parseGlobalOptions(argv []string) globalOptions {
	opts := defaultGlobalOptions()
	global := newGlobalFlagSet(&opts)
	parseGlobalFlagSet(global, argv)
	return applyParsedArgs(opts, extractParsedArgs(global))
}
