package main

func dispatchManagementRoutes(c *cli, args []string, noAuth bool) (bool, error) {
	switch args[0] {
	case "easy":
		return true, cmdEasy(c, dispatchTailArgs(args), noAuth)
	case "antiddos":
		return true, cmdAntiDDoS(c, args[1:], noAuth)
	case "ban":
		return true, cmdBanLike(c, "ban", args[1:], noAuth)
	case "unban":
		return true, cmdBanLike(c, "unban", args[1:], noAuth)
	case "bans":
		return true, cmdBans(c, args[1:], noAuth)
	case "revisions":
		return true, cmdRevisions(c, args[1:], noAuth)
	case "reports":
		return true, cmdReports(c, args[1:], noAuth)
	case "compile":
		return true, cmdRevisions(c, []string{"compile"}, noAuth)
	case "apply":
		return true, cmdRevisions(c, append([]string{"apply"}, args[1:]...), noAuth)
	case "api":
		return true, cmdAPI(c, args[1:], noAuth)
	case "maintenance":
		return true, cmdMaintenance(c, args[1:], noAuth)
	default:
		return false, nil
	}
}
