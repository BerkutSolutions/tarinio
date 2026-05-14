package main

func executeCLI(tool *cli, opts globalOptions) error {
	return dispatch(tool, opts.args, opts.noAuth)
}
