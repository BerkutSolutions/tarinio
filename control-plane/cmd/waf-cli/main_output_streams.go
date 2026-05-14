package main

import "os"

func stdoutWriter() *os.File {
	return os.Stdout
}

func stderrWriter() *os.File {
	return os.Stderr
}
