package main

func copyHeaders(columns []string) []string {
	headers := make([]string, len(columns))
	for i := range columns {
		headers[i] = columns[i]
	}
	return headers
}
