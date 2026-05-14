package main

import (
	"fmt"
	"os"
)

func readFileBytes(path string) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", path, err)
	}
	return raw, nil
}
