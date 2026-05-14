package main

func buildCLI(opts globalOptions) (*cli, error) {
	httpClient, err := newHTTPClient(opts.insecure)
	if err != nil {
		return nil, err
	}
	return newCLI(normalizeBaseURL(opts.baseURL), opts.username, opts.password, opts.outputJSON, httpClient), nil
}
