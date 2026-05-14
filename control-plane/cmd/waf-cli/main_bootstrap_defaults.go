package main

func defaultGlobalOptions() globalOptions {
	return globalOptions{
		baseURL:    envOrDefault("WAF_CLI_BASE_URL", defaultBaseURL),
		username:   defaultCLIUsername(),
		password:   defaultCLIPassword(),
		insecure:   false,
		noAuth:     false,
		outputJSON: false,
		args:       nil,
	}
}
