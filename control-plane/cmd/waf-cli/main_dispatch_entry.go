package main

func dispatch(c *cli, args []string, noAuth bool) error {
	if handled, err := dispatchByRouteGroups(c, args, noAuth); handled {
		return err
	}
	return dispatchUnknownFallback(args)
}
