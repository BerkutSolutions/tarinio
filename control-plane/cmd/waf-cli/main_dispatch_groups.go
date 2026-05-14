package main

func dispatchByRouteGroups(c *cli, args []string, noAuth bool) (bool, error) {
	if handled, err := dispatchCoreRoutes(c, args, noAuth); handled {
		return true, err
	}
	if handled, err := dispatchSecurityRoutes(c, args, noAuth); handled {
		return true, err
	}
	if handled, err := dispatchManagementRoutes(c, args, noAuth); handled {
		return true, err
	}
	return false, nil
}
