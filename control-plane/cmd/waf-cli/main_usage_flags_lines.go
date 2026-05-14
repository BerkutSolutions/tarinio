package main

func printUsageGlobalFlagLines() {
	printUsageStderrLines([]string{
		"  --base-url http://127.0.0.1:8080",
		"  --username admin",
		"  --password admin",
		"  --insecure",
		"  --no-auth",
		"  --json",
	})
}
