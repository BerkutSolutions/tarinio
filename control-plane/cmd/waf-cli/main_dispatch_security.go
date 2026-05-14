package main

func dispatchSecurityRoutes(c *cli, args []string, noAuth bool) (bool, error) {
	switch args[0] {
	case "events":
		return true, cmdEvents(c, dispatchTailArgs(args), noAuth)
	case "audit":
		return true, cmdAudit(c, args[1:], noAuth)
	case "access-policies":
		return true, cmdList(c, "access policies", "/api/access-policies", args[1:], noAuth)
	case "waf-policies":
		return true, cmdList(c, "waf policies", "/api/waf-policies", args[1:], noAuth)
	case "rate-limit-policies":
		return true, cmdList(c, "rate-limit policies", "/api/rate-limit-policies", args[1:], noAuth)
	default:
		return false, nil
	}
}
