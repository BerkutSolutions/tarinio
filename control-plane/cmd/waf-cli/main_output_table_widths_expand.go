package main

func expandColumnWidths(width []int, headers []string, rows [][]string) {
	for _, row := range rows {
		for i := 0; i < len(headers) && i < len(row); i++ {
			if len(row[i]) > width[i] {
				width[i] = len(row[i])
			}
		}
	}
}
