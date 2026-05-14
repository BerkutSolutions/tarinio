package main

import (
	"fmt"
	"io"
)

func writeFatal(writer io.Writer, format string, args ...any) {
	fmt.Fprintf(writer, "waf-cli: "+format+"\n", args...)
}
