package main

func tableCellValue(cols []string, index int) string {
	if index < len(cols) {
		return cols[index]
	}
	return ""
}
