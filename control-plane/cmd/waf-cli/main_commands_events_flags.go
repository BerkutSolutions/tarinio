package main

import "flag"

type eventsQuery struct {
	limit     int
	eventType string
	siteID    string
	severity  string
}

func parseEventsQuery(args []string) eventsQuery {
	query := eventsQuery{limit: 20}
	fs := flag.NewFlagSet("events", flag.ExitOnError)
	fs.IntVar(&query.limit, "limit", 20, "max events")
	fs.StringVar(&query.eventType, "type", "", "filter by event type")
	fs.StringVar(&query.siteID, "site-id", "", "filter by site id")
	fs.StringVar(&query.severity, "severity", "", "filter by severity")
	fs.Parse(args)
	return query
}
