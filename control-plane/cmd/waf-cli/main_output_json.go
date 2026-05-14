package main

func printJSON(value any) error {
	enc := newIndentedJSONEncoder(stdoutWriter())
	return enc.Encode(value)
}

func fatalf(format string, args ...any) {
	writeFatal(stderrWriter(), format, args...)
	exitWithErrorCode()
}
