package main

import (
	"encoding/json"
	"io"
)

func newIndentedJSONEncoder(writer io.Writer) *json.Encoder {
	enc := json.NewEncoder(writer)
	enc.SetIndent("", "  ")
	return enc
}
