package main

func runWafCLIWithArgs(argv []string) error {
	opts := runStepParse(argv)
	return runStepRun(opts)
}
