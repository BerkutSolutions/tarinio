package main

func printUsageStderrLines(lines []string) {
	for _, line := range lines {
		printUsageStderrLine(line)
	}
}
