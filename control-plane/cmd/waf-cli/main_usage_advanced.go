package main

func printUsageAdvancedCommands() {
	printUsageStderrLines([]string{
		"  easy get <site-id> | easy upsert <site-id> --file profile.json",
		"  antiddos get | antiddos upsert --file antiddos.json",
		"  revisions compile | revisions apply <rev-id>",
		"  reports revisions",
		"  api <GET|POST|PUT|DELETE> <path> [--file body.json]",
	})
}
