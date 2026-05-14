package main

func ensureCommandArgs(opts globalOptions) error {
	if len(opts.args) == 0 {
		usage()
		return errCommandRequired()
	}
	return nil
}
