package main

func defaultCLIUsername() string {
	return envOrDefault("WAF_CLI_USERNAME", envOrDefault("CONTROL_PLANE_BOOTSTRAP_ADMIN_USERNAME", "admin"))
}

func defaultCLIPassword() string {
	return envOrDefault("WAF_CLI_PASSWORD", envOrDefault("CONTROL_PLANE_BOOTSTRAP_ADMIN_PASSWORD", "admin"))
}
