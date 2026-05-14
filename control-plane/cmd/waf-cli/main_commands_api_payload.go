package main

import "strings"

func loadAPIPayload(file string) (any, error) {
	if strings.TrimSpace(file) == "" {
		return nil, nil
	}
	return readJSONFile(file)
}
