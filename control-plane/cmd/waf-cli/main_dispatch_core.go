package main

func dispatchCoreRoutes(c *cli, args []string, noAuth bool) (bool, error) {
	switch args[0] {
	case "health":
		return true, cmdHealth(c)
	case "setup":
		return true, cmdSetup(c, dispatchTailArgs(args), noAuth)
	case "me":
		return true, cmdAuthMe(c, noAuth)
	case "sites":
		return true, cmdSites(c, args[1:], noAuth)
	case "upstreams":
		return true, cmdList(c, "upstreams", "/api/upstreams", args[1:], noAuth)
	case "tls":
		return true, cmdList(c, "tls configs", "/api/tls-configs", args[1:], noAuth)
	case "certificates":
		return true, cmdList(c, "certificates", "/api/certificates", args[1:], noAuth)
	default:
		return false, nil
	}
}
