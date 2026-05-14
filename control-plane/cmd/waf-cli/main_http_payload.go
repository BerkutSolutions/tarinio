package main

import (
	"bytes"
	"encoding/json"
	"io"
)

func buildJSONBodyReader(payload any) (io.Reader, error) {
	if payload == nil {
		return nil, nil
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(raw), nil
}
