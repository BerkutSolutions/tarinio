package main

func runWithOptions(opts globalOptions) error {
	if err := runStepValidate(opts); err != nil {
		return err
	}
	tool, err := runStepInit(opts)
	if err != nil {
		return err
	}
	if err := runStepExecute(tool, opts); err != nil {
		return err
	}
	return nil
}
