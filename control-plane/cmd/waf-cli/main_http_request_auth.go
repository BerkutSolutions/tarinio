package main

func ensureRawRequestAuth(c *cli, auth bool) error {
	return ensureRequestAuth(c, auth)
}
