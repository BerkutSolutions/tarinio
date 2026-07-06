package main

func printUsageCoreCommands() {
	printUsageStderrLines([]string{
		"Core commands:",
		"  health",
		"  setup status",
		"  me",
		"  sites list | sites delete <id>",
		"  upstreams list",
		"  tls list",
		"  certificates list",
		"  events [--limit N]",
		"  audit [--action A --site-id S --status ST --limit N --offset N]",
		"  ban <ip> [--site control-plane-access]",
		"  unban <ip> [--site control-plane-access]",
		"  bans list [--site control-plane-access]",
		"  maintenance",
		"  access-policies list",
		"  waf-policies list",
		"  rate-limit-policies list",
	})
}
