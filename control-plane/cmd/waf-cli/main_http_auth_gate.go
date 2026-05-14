package main

func ensureRequestAuth(c *cli, auth bool) error {
	if auth {
		return c.ensureLogin()
	}
	return nil
}
