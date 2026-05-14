package main

func printSites(c *cli, value any) error {
	if c.outputJSON {
		return printJSON(value)
	}
	items := asList(value)
	renderSitesHeader(len(items))
	if len(items) == 0 {
		return nil
	}
	return printTableFromMaps(items, defaultSitesColumns())
}

func printKV(pairs ...string) {
	for i := 0; i+1 < len(pairs); i += 2 {
		printKVLine(pairs[i], pairs[i+1])
	}
}
