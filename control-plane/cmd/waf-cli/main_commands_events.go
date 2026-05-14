package main

import "net/http"

func cmdEvents(c *cli, args []string, noAuth bool) error {
	query := parseEventsQuery(args)

	value, err := c.requestJSON(http.MethodGet, "/api/events", nil, !noAuth)
	if err != nil {
		return err
	}
	root, ok := value.(map[string]any)
	if !ok {
		return printJSON(value)
	}
	items := asList(root["events"])
	filtered := filterEvents(items, query)
	return renderEvents(c, filtered)
}
