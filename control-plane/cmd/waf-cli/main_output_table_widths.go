package main

func initColumnWidths(headers []string) []int {
	width := make([]int, len(headers))
	for i := range headers {
		width[i] = len(headers[i])
	}
	return width
}
