package main

import "fmt"

func initCLI(opts globalOptions) (*cli, error) {
	tool, err := buildCLI(opts)
	if err != nil {
		return nil, fmt.Errorf("init http client: %w", err)
	}
	return tool, nil
}
