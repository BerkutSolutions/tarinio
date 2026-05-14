package main

import "net/url"

func siteDeletePath(id string) string {
	return "/api/sites/" + url.PathEscape(id)
}
