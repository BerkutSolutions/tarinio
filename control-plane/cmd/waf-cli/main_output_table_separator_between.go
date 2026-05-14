package main

func tableBetweenColumns(index int, total int) string {
	if index != total-1 {
		return "  "
	}
	return ""
}
